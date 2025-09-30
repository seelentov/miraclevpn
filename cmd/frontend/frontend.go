package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	"miraclevpn/internal/http/controller"
	"miraclevpn/internal/http/middleware"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/cookie"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/payment"
	"miraclevpn/internal/services/user"
	"miraclevpn/pkg/yookassa"
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

	dbUser := os.Getenv("DB_USER")
	dbHost := os.Getenv("DB_HOST")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbSsl := os.Getenv("DB_SSLMODE")
	dbTZ := os.Getenv("DB_TIMEZONE")

	paymentExpirationStr := os.Getenv("PAYMENT_EXPIRATION_SEC")
	paymentExpiration, err := strconv.Atoi(paymentExpirationStr)

	if err != nil {
		log.Fatal("failed get PAYMENT_EXPIRATION_SEC: " + err.Error())
	}

	jwtSecretPayment := os.Getenv("JWT_SECRET_PAYMENT")

	logger, err := logg.NewZapLogger("", 0, debug)
	if err != nil {
		log.Fatal(err)
	}

	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ, "MIIVPN_API")
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	userRepo := repo.NewUserRepository(gormDB, nil, 0)
	payRepo := repo.NewPaymentRepository(gormDB, time.Second*time.Duration(paymentExpiration))
	payPlRepo := repo.NewPaymentPlanRepository(gormDB)

	paymentClient := yookassa.NewClient(os.Getenv("PAYMENT_SHOP_ID"), os.Getenv("PAYMENT_SECRET"), os.Getenv("PAYMENT_RETURN_URL"))

	jwtPaySrv := crypt.NewJwtService(jwtSecretPayment, logger.Logger)
	paySrv := payment.NewPaymentService(paymentClient, payRepo, payPlRepo, jwtPaySrv, logger.Logger)
	userSrv := user.NewUserService(userRepo, logger.Logger)
	jwtSrv := crypt.NewJwtService(jwtSecretAuth, logger.Logger)
	cookieSrv := cookie.NewCookieService(domain)

	viewCtrl := controller.NewViewIndexController()
	authCtrl := controller.NewViewAuthController(cookieSrv, userSrv)
	payCtrl := controller.NewViewPaymentController(paySrv, userSrv)

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
	}

	if err := setupStatic(r); err != nil {
		log.Fatalf("Error registering static files: %v", err)
	}

	r.Run(":" + os.Getenv("PORT_FRONTEND"))
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
