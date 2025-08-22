package main

import (
	"log"
	"math"
	"miraclevpn/internal/daemon"
	"miraclevpn/internal/daemon/healthcheck"
	"miraclevpn/internal/http/middleware"
	"miraclevpn/internal/models"
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
	"miraclevpn/pkg/ovpn"
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

	sshUser := os.Getenv("SSH_USER")
	sshStatusPath := os.Getenv("SSH_STATUS_PATH")
	sshCreateUserFile := os.Getenv("SSH_CREATE_USER_FILE")
	sshRevokeUserFile := os.Getenv("SSH_REVOKE_USER_FILE")

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

	// VPN
	vpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile)

	// Сервисы
	authSrv := auth.NewAuthService(userRepo, veriRepo, tgSrv, jwtSrv, time.Duration(jwtDuration)*time.Minute, logger.Logger)
	userSrv := user.NewUserService(userRepo, veriRepo, tgSrv, logger.Logger)
	serversSrv := servers.NewServersService(userServerRepo, serverRepo, userRepo, vpnSrv, logger.Logger)

	// Контроллеры
	authCtrl := controller.NewAuthController(authSrv, jwtSrv)
	userCtrl := controller.NewUserController(userSrv)
	serverCtrl := controller.NewServerController(serversSrv)

	//Демоны
	tgDaemon := daemon.NewTgDaemon(tgToken, jwtSrv, userRepo, logger.Logger)
	tgDaemon.Start()
	defer tgDaemon.Stop()

	healthCheckIntervalSec := 60
	h := os.Getenv("HEALTHCHECK_INTERVAL_SEC")
	if h != "" {
		healthCheckIntervalSec, err = strconv.Atoi(h)
		if err != nil || healthCheckIntervalSec <= 0 {
			logger.Logger.Error("invalid HEALTHCHECK_INTERVAL_SEC, using default 5 seconds", zap.Error(err))
		}
	}

	healthCheckDuration := time.Second * time.Duration(healthCheckIntervalSec)

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewTgClient(tgTokenHealthCheck, "")

	dbHealthCheck := healthcheck.NewDBHealthCheck(gormDB, healthCheckDuration, logger.Logger, tgSenderHealthCheck, tgChatIDHealthCheck)
	dbHealthCheck.Start()
	defer dbHealthCheck.Stop()

	vpnHealthCheck := healthcheck.NewVpnHealthCheck(healthCheckDuration, logger.Logger, vpnSrv, serverRepo, tgSenderHealthCheck, tgChatIDHealthCheck)
	vpnHealthCheck.Start()
	defer vpnHealthCheck.Stop()

	tgHealthCheck := healthcheck.NewTgHealthCheck(healthCheckDuration, logger.Logger, tgSenderHealthCheck, tgChatIDHealthCheck)
	tgHealthCheck.Start()
	defer tgHealthCheck.Stop()

	r := gin.Default()
	r.Use(middleware.Recovery(debug, tgSenderHealthCheck, tgChatIDHealthCheck, logger.Logger))
	r.NoRoute(middleware.NotFound())
	r.Use(middleware.SetUserIDMiddleware(jwtSrv))

	api := r.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			auth := v1.Group("/auth")
			{
				auth.POST("/login", authCtrl.PostLogin)
				auth.POST("/register", authCtrl.PostRegister)

				refresh := auth.Group("/refresh")
				refresh.Use(middleware.RequireAuthMiddleware(userRepo))
				{
					refresh.POST("/", authCtrl.PostRefresh)
				}
			}

			security := v1.Group("/security")
			{
				security.POST("/try-change-password", userCtrl.PostChangePasswordSend)
				security.POST("/change-password", userCtrl.PostChangePasswordVerify)
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
					serverGroup.GET("/:id", serverCtrl.GetServer)
				}
			}
		}
	}

	//DEBUG
	if debug {
		p, err := argonSrv.GenerateHash("12345678")
		if err != nil {
			logger.Logger.Fatal("cant create debug user", zap.Error(err))
		}

		gormDB.Save(&models.User{
			ID:        1,
			Username:  "testuser",
			Password:  p,
			TGChat:    nil,
			Active:    false,
			ExpiredAt: time.Now().Add(time.Hour * 24 * 365),
		})

		token, err := jwtSrv.GenerateToken("1", time.Duration(jwtDuration)*time.Minute)
		if err != nil {
			logger.Logger.Fatal("cant generate debug token", zap.Error(err))
		}
		logger.Logger.Info("debug token", zap.String("token", token))
	}

	r.Run(":" + os.Getenv("PORT"))
}
