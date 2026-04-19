package main

import (
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	"miraclevpn/internal/controller/http/controller"
	"miraclevpn/internal/controller/http/middleware"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/auth"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/info"
	"miraclevpn/internal/services/payment"
	"miraclevpn/internal/services/servers"
	"miraclevpn/internal/services/user"
	vpnrouter "miraclevpn/internal/services/vpn"
	"miraclevpn/pkg/awg"
	"miraclevpn/pkg/ovpn"
	"miraclevpn/pkg/tg"
	"miraclevpn/pkg/yookassa"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"go.uber.org/zap"

	"net/http"
	_ "net/http/pprof"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	dbUser := os.Getenv("DB_USER")
	dbHost := os.Getenv("DB_HOST")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbSsl := os.Getenv("DB_SSLMODE")
	dbTZ := os.Getenv("DB_TIMEZONE")
	logDir := os.Getenv("LOG_DIR")
	logRetain, _ := strconv.Atoi(os.Getenv("LOG_RETAIN"))
	debug := os.Getenv("DEBUG") == "true"
	jwtSecretAuth := os.Getenv("JWT_SECRET_AUTH")
	jwtSecretPayment := os.Getenv("JWT_SECRET_PAYMENT")

	sshUser := os.Getenv("SSH_USER")
	sshStatusPath := os.Getenv("SSH_STATUS_PATH")
	sshCreateUserFile := os.Getenv("SSH_CREATE_USER_FILE")
	sshRevokeUserFile := os.Getenv("SSH_REVOKE_USER_FILE")
	sshConfigsDir := os.Getenv("SSH_CONFIGS_DIR")

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

	proofKey := os.Getenv("MII_VPN_PROOF")
	proofBanIfFail := os.Getenv("PROOF_BAN_IF_FAIL") == "true"
	proofKeys := make(map[string]string)

	if proofKey != "" {
		log.Println(proofKey[:3] + "***" + proofKey[len(proofKey)-3:])
		for _, kv := range strings.Split(proofKey, "::") {
			kvv := strings.Split(kv, ":")
			if len(kvv) >= 2 {
				proofKeys[kvv[0]] = kvv[1]
				log.Println(kvv[0] + ":" + kvv[1][:3] + "***" + kvv[1][len(kvv[1])-3:])
			}
		}

		if len(proofKeys) == 0 {
			log.Println("PROOF KEY PROVIDED BUT INVALID FORMAT")
		}
	}

	vpnConfigExpirationStt := os.Getenv("VPN_CONFIG_DIRATION_SEC")

	vpnConfigExpiration, err := strconv.Atoi(vpnConfigExpirationStt)
	if err != nil {
		log.Fatal("failed get VPN_CONFIG_DIRATION_SEC: " + err.Error())
	}

	freeTrial, err := strconv.Atoi(os.Getenv("FREE_TRIAL_SEC"))
	if err != nil {
		log.Fatal("failed get FREE_TRIAL_SEC: " + err.Error())
	}

	jwtDuration := math.MaxInt32
	if os.Getenv("JWT_DURATION_MIN") != "" && os.Getenv("JWT_DURATION_MIN") != "0" {
		jwtDuration, _ = strconv.Atoi(os.Getenv("JWT_DURATION_MIN"))
	}

	paymentExpirationStr := os.Getenv("PAYMENT_EXPIRATION_SEC")
	paymentExpiration, err := strconv.Atoi(paymentExpirationStr)

	if err != nil {
		log.Fatal("failed get PAYMENT_EXPIRATION_SEC: " + err.Error())
	}

	logger, err := logg.NewZapLogger(logDir, logRetain, debug)
	if err != nil {
		log.Fatal(err)
	}

	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ, "MIIVPN_API")
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	if os.Getenv("PPROF_PORT") != "" {
		go func() {
			logger.Logger.Info("Open pprof on :" + os.Getenv("PPROF_PORT"))
			log.Println(http.ListenAndServe(":"+os.Getenv("PPROF_PORT"), nil))
		}()
	}

	argonParams := &crypt.Argon2idParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
	argonSrv := crypt.NewArgonService(argonParams, logger.Logger)
	jwtSrv := crypt.NewJwtService(jwtSecretAuth, logger.Logger)
	jwtPaySrv := crypt.NewJwtService(jwtSecretPayment, logger.Logger)

	userRepo := repo.NewUserRepository(gormDB, argonSrv, time.Duration(freeTrial)*time.Second)
	serverRepo := repo.NewServerRepository(gormDB)
	userServerRepo := repo.NewUserServerRepository(gormDB)
	newsRepo := repo.NewNewsRepository(gormDB)
	infoRepo := repo.NewInfoRepository(gormDB)
	keyValueRepo := repo.NewKeyValueRepository(gormDB)
	payPlRepo := repo.NewPaymentPlanRepository(gormDB)
	authDataRepo := repo.NewAuthDataRepository(gormDB)
	payRepo := repo.NewPaymentRepository(gormDB, time.Second*time.Duration(paymentExpiration))

	ovpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)
	awgSrv := awg.NewClient(awgSSHUser, awgManageScript, awgClientsDir)
	vpnSrv := vpnrouter.NewVpnRouter(ovpnSrv, awgSrv, serverRepo)

	paymentClient := yookassa.NewClient(os.Getenv("PAYMENT_SHOP_ID"), os.Getenv("PAYMENT_SECRET"), os.Getenv("PAYMENT_RETURN_URL"))

	authSrv := auth.NewAuthService(userRepo, authDataRepo, jwtSrv, time.Duration(jwtDuration)*time.Minute, logger.Logger)
	userSrv := user.NewUserService(userRepo, logger.Logger)
	serversSrv := servers.NewServersService(userServerRepo, serverRepo, userRepo, vpnSrv, logger.Logger)
	infoSrv := info.NewInfoService(newsRepo, infoRepo, keyValueRepo, payPlRepo)
	paySrv := payment.NewPaymentService(paymentClient, payRepo, payPlRepo, jwtPaySrv, logger.Logger)

	authCtrl := controller.NewAuthController(authSrv, jwtSrv, jwtDuration)
	userCtrl := controller.NewUserController(userSrv)
	serverCtrl := controller.NewServerController(serversSrv, vpnConfigExpiration)
	infoCtrl := controller.NewInfoController(infoSrv)
	payCtrl := controller.NewPaymentController(paySrv, userSrv)

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewClient(tgTokenHealthCheck, "")

	r := gin.Default()

	if debug {
		r.SetTrustedProxies(nil)
	} else {
		r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}

	r.Static("/storage", "./storage")

	r.Use(middleware.Recovery(debug, tgSenderHealthCheck, tgChatIDHealthCheck, logger.Logger))
	r.NoRoute(middleware.NotFound())

	api := r.Group("/api")
	{
		api.GET("/ping", infoCtrl.GetPing)

		api.POST("/echo", infoCtrl.PostEcho)

		v1 := api.Group("/v1")
		{
			v1.Use(middleware.SetUserIDMiddleware(jwtSrv))

			payment := v1.Group("/payment")
			{
				payment.POST("/hook", payCtrl.PostPaymentHook)
				payment.POST("/create", payCtrl.PostCreate)
				payment.POST("/remove", payCtrl.PostRemovePaymentMethod)
			}

			if len(proofKeys) > 0 {
				log.Println("PROOF ACTIVATED")
				v1.Use(middleware.ProofMiddleware(proofKeys, proofBanIfFail, debug))
			}

			auth := v1.Group("/auth")
			{
				auth.POST("/login", authCtrl.PostLogin)
				refresh := auth.Group("/refresh")
				refresh.Use(middleware.RequireAuthMiddleware(userRepo))
				{
					refresh.POST("/", authCtrl.PostRefresh)
				}
			}

			o := v1.Group("/")
			o.Use(middleware.RequireAuthMiddleware(userRepo))
			{
				userGroup := o.Group("/user")
				{
					userGroup.GET("/", userCtrl.GetUser)
				}
				serverGroup := o.Group("/server")
				{
					serverGroup.GET("/", serverCtrl.GetServers)
					serverGroup.GET("/region", serverCtrl.GetRegions)
					serverGroup.GET("/region/:region", serverCtrl.GetServersByRegion)
					serverGroup.GET("/:id", middleware.CheckUserMiddleware(userRepo), serverCtrl.GetServer)
					serverGroup.GET("/status/:id", serverCtrl.GetServerStatus)
					serverGroup.GET("/status/region/:region", serverCtrl.GetRegionStatus)

					serverGroup.GET("/preview", serverCtrl.GetPreview)
					serverGroup.POST("/preview", serverCtrl.PostRequest)
				}
				info := o.Group("/info")
				{
					info.GET("", infoCtrl.GetInfos)
					info.GET("/support", infoCtrl.GetSupport)
					info.GET("/news", infoCtrl.GetNews)
					info.GET("/tech_work", infoCtrl.GetTechWork)
				}
			}
			i := v1.Group("/info")
			{
				i.GET("/payment", infoCtrl.GetPaymentPlans)
				i.GET("/payment/:plan_id", infoCtrl.GetPaymentPlan)
				i.GET("/:slug", infoCtrl.GetInfo)
			}
		}
	}

	r.Run(":" + os.Getenv("PORT"))

	select {}
}
