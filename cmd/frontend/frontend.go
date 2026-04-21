package main

import (
	"errors"
	"log"
	"math"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	"miraclevpn/internal/controller/http/controller"
	"miraclevpn/internal/controller/http/middleware"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/cookie"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/payment"
	"miraclevpn/internal/services/servers"
	"miraclevpn/internal/services/user"
	vpnrouter "miraclevpn/internal/services/vpn"
	"miraclevpn/pkg/awg"
	"miraclevpn/pkg/ovpn"
	"miraclevpn/pkg/yookassa"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	debug := os.Getenv("DEBUG") == "true"
	jwtSecretAuth := os.Getenv("JWT_SECRET_AUTH")
	domain := os.Getenv("DOMAIN")

	paymentExpirationStr := os.Getenv("PAYMENT_EXPIRATION_SEC")
	paymentExpiration, err := strconv.Atoi(paymentExpirationStr)
	if err != nil {
		log.Fatal("failed get PAYMENT_EXPIRATION_SEC: " + err.Error())
	}

	freeTrial, err := strconv.Atoi(os.Getenv("FREE_TRIAL_SEC"))
	if err != nil {
		log.Fatal("failed get FREE_TRIAL_SEC: " + err.Error())
	}

	jwtDuration := math.MaxInt32
	if v := os.Getenv("JWT_DURATION_MIN"); v != "" && v != "0" {
		jwtDuration, _ = strconv.Atoi(v)
	}

	jwtSecretPayment := os.Getenv("JWT_SECRET_PAYMENT")

	logger, err := logg.NewZapLogger("", 0, debug)
	if err != nil {
		log.Fatal(err)
	}

	gormDB, err := db.NewConnFromEnv()
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	sshUser := os.Getenv("OVPN_SSH_USER")
	ovpnSrv := ovpn.NewClient(
		sshUser,
		os.Getenv("OVPN_STATUS_PATH"),
		os.Getenv("OVPN_CREATE_USER_FILE"),
		os.Getenv("OVPN_REVOKE_USER_FILE"),
		os.Getenv("OVPN_CONFIGS_DIR"),
	)
	awgSSHUser := os.Getenv("AWG_SSH_USER")
	if awgSSHUser == "" {
		awgSSHUser = sshUser
	}
	awgManageScript := os.Getenv("AWG_MANAGE_SCRIPT")
	if awgManageScript == "" {
		awgManageScript = "/usr/local/bin/wg-manage.sh"
	}
	awgClientsDir := os.Getenv("AWG_CLIENTS_DIR")
	if awgClientsDir == "" {
		awgClientsDir = "/etc/wireguard/clients"
	}
	awgSrv := awg.NewClient(awgSSHUser, awgManageScript, awgClientsDir)

	userRepo := repo.NewUserRepository(gormDB, nil, time.Duration(freeTrial)*time.Second)
	userServerRepo := repo.NewUserServerRepository(gormDB)
	serverRepo := repo.NewServerRepository(gormDB)
	reviewRepo := repo.NewReviewRepository(gormDB)
	payRepo := repo.NewPaymentRepository(gormDB, time.Second*time.Duration(paymentExpiration))
	payPlRepo := repo.NewPaymentPlanRepository(gormDB)

	paymentClient := yookassa.NewClient(os.Getenv("PAYMENT_SHOP_ID"), os.Getenv("PAYMENT_SECRET"), os.Getenv("PAYMENT_RETURN_URL"))

	vpnRouter := vpnrouter.NewVpnRouter(ovpnSrv, awgSrv, serverRepo)
	serversSrv := servers.NewServersService(userServerRepo, serverRepo, userRepo, vpnRouter, logger.Logger)

	jwtPaySrv := crypt.NewJwtService(jwtSecretPayment, logger.Logger)
	paySrv := payment.NewPaymentService(paymentClient, payRepo, payPlRepo, jwtPaySrv, logger.Logger)
	userSrv := user.NewUserService(userRepo, logger.Logger)
	jwtSrv := crypt.NewJwtService(jwtSecretAuth, logger.Logger)
	cookieSrv := cookie.NewCookieService(domain)

	paymentURL := os.Getenv("PAYMENT_URL")
	lkURL := os.Getenv("LK_URL")

	viewCtrl := controller.NewViewIndexController(reviewRepo)
	authCtrl := controller.NewViewAuthController(cookieSrv, userSrv)
	payCtrl := controller.NewViewPaymentController(paySrv, userSrv)
	chatCtrl := controller.NewChatController(userRepo, serversSrv, jwtSrv, time.Duration(jwtDuration)*time.Minute, paymentURL, lkURL, logger.Logger)

	r := gin.Default()
	r.LoadHTMLGlob("./templates/*.html")
	if debug {
		r.SetTrustedProxies(nil)
	} else {
		r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}

	r.NoRoute(viewCtrl.NotFound)
	r.Use(middleware.AuthCookie(jwtSrv, cookieSrv))
	r.Use(gin.RecoveryWithWriter(gin.DefaultErrorWriter, viewCtrl.Panic))

	r.GET("/api/regions", publicRegionsHandler(serverRepo))
	r.POST("/api/server/request", publicServerRequestHandler(serverRepo))

	r.GET("/chat", chatCtrl.GetPage)
	r.POST("/api/chat/start", chatCtrl.Start)
	r.POST("/api/chat/action", chatCtrl.Action)
	r.GET("/api/chat/dl/:server_id", chatCtrl.GetConfig)

	r.GET("/", viewCtrl.GetIndex)
	r.GET("/success-payment", viewCtrl.GetSuccessPayment)

	r.GET("/login", authCtrl.GetLogin)
	r.POST("/login", authCtrl.PostLogin)

	r.GET("/payments", payCtrl.GetPayments)

	onlyAuth := r.Group("/user")
	onlyAuth.Use(middleware.AuthReqFrontend(userRepo))
	{
		onlyAuth.GET("/", authCtrl.GetLK)
		onlyAuth.POST("/payment", payCtrl.PostPayment)
		onlyAuth.POST("/remove-payment", payCtrl.PostRemovePaymentMethod)
		onlyAuth.POST("/logout", authCtrl.PostLogout)
	}

	if err := setupStatic(r); err != nil {
		log.Fatalf("Error registering static files: %v", err)
	}

	r.Run(":" + os.Getenv("PORT_FRONTEND"))
}

type publicServer struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type publicRegion struct {
	Code    string         `json:"code"`
	Name    string         `json:"name"`
	FlagURL string         `json:"flag_url"`
	Lat     float64        `json:"lat"`
	Lng     float64        `json:"lng"`
	Servers []publicServer `json:"servers"`
}

func publicRegionsHandler(srvRepo *repo.ServerRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		servers, err := srvRepo.FindAllForMap()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		regionMap := map[string]*publicRegion{}
		var regionOrder []string

		for _, srv := range servers {
			if _, exists := regionMap[srv.Region]; !exists {
				regionMap[srv.Region] = &publicRegion{
					Code:    srv.Region,
					Name:    srv.RegionName,
					FlagURL: srv.RegionFlagURL,
					Servers: []publicServer{},
				}
				regionOrder = append(regionOrder, srv.Region)
			}
			r := regionMap[srv.Region]
			if srv.Lat != 0 || srv.Lng != 0 {
				r.Lat = srv.Lat
				r.Lng = srv.Lng
			}
			if !srv.Preview {
				r.Servers = append(r.Servers, publicServer{
					Name: srv.Name,
					Type: srv.Type,
				})
			}
		}

		result := make([]*publicRegion, 0, len(regionOrder))
		for _, code := range regionOrder {
			result = append(result, regionMap[code])
		}

		ctx.JSON(http.StatusOK, result)
	}
}

func publicServerRequestHandler(srvRepo *repo.ServerRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req struct {
			Region string `json:"region" binding:"required"`
		}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "region required"})
			return
		}

		userID := "anon:" + ctx.ClientIP()
		if err := srvRepo.SendRequest(req.Region, userID); err != nil {
			if errors.Is(err, repo.ErrReqAlreadyExist) {
				ctx.JSON(http.StatusOK, gin.H{"status": "already_requested"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func setupStatic(r *gin.Engine) error {
	publicDir := "./public"

	var files []string
	var dirs []string

	err := filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(publicDir, path)
		if err != nil {
			return err
		}

		if relativePath == "." {
			return nil
		}

		webPath := "/" + filepath.ToSlash(relativePath)

		if info.IsDir() {
			dirs = append(dirs, webPath)
		} else {
			files = append(files, webPath)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		isSubdir := false
		for _, other := range dirs {
			if other != dir && strings.HasPrefix(dir, other+"/") {
				isSubdir = true
				break
			}
		}
		if isSubdir {
			continue
		}

		hasFilesInDir := false
		for _, file := range files {
			if strings.HasPrefix(file, dir+"/") {
				hasFilesInDir = true
				break
			}
		}

		if hasFilesInDir {
			log.Printf("Registering directory: %s -> %s%s", dir, publicDir, dir)
			r.Static(dir, publicDir+dir)
		}
	}

	for _, file := range files {
		inRegisteredDir := false
		for _, dir := range dirs {
			if strings.HasPrefix(file, dir+"/") {
				inRegisteredDir = true
				break
			}
		}

		if !inRegisteredDir {
			log.Printf("Registering file: %s -> %s%s", file, publicDir, file)
			r.StaticFile(file, publicDir+file)
		}
	}

	return nil
}
