package main

import (
	"log"
	"math"
	"miraclevpn/internal/http/middleware"
	"os"
	"strconv"
	"time"

	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	"miraclevpn/internal/http/controller"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/auth"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/services/servers"
	"miraclevpn/internal/services/user"
	"miraclevpn/internal/services/vpn"
	"miraclevpn/pkg/tg"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	// Получение параметров из .env
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
	jwtSecret := os.Getenv("JWT_SECRET")

	jwtDuration := math.MaxInt32
	if os.Getenv("JWT_DURATION_MIN") != "" && os.Getenv("JWT_DURATION_MIN") != "0" {
		jwtDuration, _ = strconv.Atoi(os.Getenv("JWT_DURATION_MIN"))
	}

	tgToken := os.Getenv("TG_TOKEN")
	tgName := os.Getenv("TG_NAME")

	// Инициализация логгера
	logger, err := logg.NewZapLogger(logDir, logRetain, debug)
	if err != nil {
		log.Fatal(err)
	}

	// Подключение к БД PostgreSQL
	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ)
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	// Криптосервисы
	argonParams := &crypt.Argon2idParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
	argonSrv := crypt.NewArgonService(argonParams, logger.Logger)
	jwtSrv := crypt.NewJwtService(jwtSecret, logger.Logger)

	// Репозитории
	userRepo := repo.NewUserRepository(gormDB, argonSrv)
	veriRepo := repo.NewVerifierRepository(gormDB)
	serverRepo := repo.NewServerRepository(gormDB)
	userServerRepo := repo.NewUserServerRepository(gormDB)

	// Telegram sender
	tgSender := tg.NewTgClient(tgToken, tgName)
	tgSrv := sender.NewTgService(userRepo, tgSender, logger.Logger)

	// VPN сервис (пример, замените на свою реализацию)
	var vpnSrv vpn.VpnService // Инициализация vpn.Client или другого vpn.VpnService

	// Сервисы
	authSrv := auth.NewAuthService(userRepo, veriRepo, tgSrv, jwtSrv, time.Duration(jwtDuration)*time.Minute)
	userSrv := user.NewUserService(userRepo, veriRepo, logger.Logger)
	serversSrv := servers.NewServersService(userServerRepo, serverRepo, userRepo, vpnSrv, logger.Logger)

	// Контроллеры
	authCtrl := controller.NewAuthController(authSrv, jwtSrv)
	userCtrl := controller.NewUserController(userSrv)
	serverCtrl := controller.NewServerController(serversSrv)

	r := gin.Default()
	r.Use(middleware.Recovery())
	r.NoRoute(middleware.NotFound())
	r.Use(middleware.RefreshTokenMiddleware(jwtSrv, time.Duration(jwtDuration)*time.Minute))
	r.Use(middleware.SetUserIDMiddleware(jwtSrv))

	api := r.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			auth := v1.Group("/auth")
			{
				auth.POST("/login", authCtrl.PostLogin)
				auth.POST("/register", authCtrl.PostRegister)
				auth.POST("/activate", authCtrl.PostActivate)
			}

			o := v1.Group("/")
			o.Use(middleware.RequireUserIDMiddleware())
			{
				userGroup := o.Group("/user")
				{
					userGroup.GET("/", userCtrl.GetUser)
				}
				serverGroup := o.Group("/server")
				{
					serverGroup.GET("/", serverCtrl.GetServers)
					serverGroup.GET("/region/:region", serverCtrl.GetServersByRegion)
					serverGroup.GET("/:id", serverCtrl.GetServer)
				}
			}
		}
	}

	r.Run(":" + os.Getenv("PORT"))
}
