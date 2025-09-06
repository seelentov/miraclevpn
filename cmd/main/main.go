package main

import (
	"log"
	"math"
	authdaemon "miraclevpn/internal/daemon/auth_daemon"
	"miraclevpn/internal/daemon/healthcheck"
	vpndaemon "miraclevpn/internal/daemon/vpn_daemon"
	"miraclevpn/internal/http/middleware"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	"miraclevpn/internal/http/controller"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/auth"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/info"
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
	sshConfigsDir := os.Getenv("SSH_CONFIGS_DIR")

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

	vpnRefreshConfigIntervalStr := os.Getenv("VPN_REFRESH_INTERVAL_SEC")
	vpnConfigExpirationStt := os.Getenv("VPN_CONFIG_DIRATION_SEC")

	vpnRefreshConfigInterval, err := strconv.Atoi(vpnRefreshConfigIntervalStr)
	if err != nil {
		log.Fatal("failed get VPN_REFRESH_INTERVAL_SEC: " + err.Error())
	}

	vpnRemoveExpiredIntervalStr := os.Getenv("VPN_REMOVE_EXPIRED_INTERVAL_SEC")
	vpnRemoveExpiredInterval, err := strconv.Atoi(vpnRemoveExpiredIntervalStr)

	if err != nil {
		log.Fatal("failed get VPN_REMOVE_EXPIRED_INTERVAL_SEC: " + err.Error())
	}

	authFindSuspiciousIntervalStr := os.Getenv("AUTH_FIND_SUSPICIOUS_INTERVAL_SEC")
	authFindSuspiciousInterval, err := strconv.Atoi(authFindSuspiciousIntervalStr)

	if err != nil {
		log.Fatal("failed get AUTH_FIND_SUSPICIOUS_INTERVAL_SEC: " + err.Error())
	}

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
	userRepo := repo.NewUserRepository(gormDB, argonSrv, time.Duration(freeTrial)*time.Second)
	serverRepo := repo.NewServerRepository(gormDB)
	userServerRepo := repo.NewUserServerRepository(gormDB)
	newsRepo := repo.NewNewsRepository(gormDB)
	infoRepo := repo.NewInfoRepository(gormDB)
	keyValueRepo := repo.NewKeyValueRepository(gormDB)
	payPlRepo := repo.NewPaymentPlanRepository(gormDB)
	authDataRepo := repo.NewAuthDataRepository(gormDB)

	// VPN
	vpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)

	// Сервисы
	authSrv := auth.NewAuthService(userRepo, authDataRepo, jwtSrv, time.Duration(jwtDuration)*time.Minute, logger.Logger)
	userSrv := user.NewUserService(userRepo, logger.Logger)
	serversSrv := servers.NewServersService(userServerRepo, serverRepo, userRepo, vpnSrv, logger.Logger)
	infoSrv := info.NewInfoService(newsRepo, infoRepo, keyValueRepo, payPlRepo)

	// Контроллеры
	authCtrl := controller.NewAuthController(authSrv, jwtSrv, jwtDuration)
	userCtrl := controller.NewUserController(userSrv)
	serverCtrl := controller.NewServerController(serversSrv, vpnConfigExpiration)
	infoCtrl := controller.NewInfoController(infoSrv)

	//Админ TG
	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewTgClient(tgTokenHealthCheck, "")

	//Демоны
	vpnRefreshDaemon := vpndaemon.NewVpnRefreshDaemon(time.Second*time.Duration(vpnRefreshConfigInterval), logger.Logger, serversSrv, tgSenderHealthCheck, tgChatIDHealthCheck, time.Second*time.Duration(vpnConfigExpiration))
	vpnRefreshDaemon.Start()
	defer vpnRefreshDaemon.Stop()

	vpnRemoveExpiredDaemon := vpndaemon.NewVpnRemoveExpiredDaemon(time.Second*time.Duration(vpnRemoveExpiredInterval), logger.Logger, serversSrv, tgSenderHealthCheck, tgChatIDHealthCheck)
	vpnRemoveExpiredDaemon.Start()
	defer vpnRemoveExpiredDaemon.Stop()

	authFindSuspiciosDaemon := authdaemon.NewAuthFindSuspicious(time.Second*time.Duration(authFindSuspiciousInterval), logger.Logger, authDataRepo, tgSenderHealthCheck, tgChatIDHealthCheck)
	authFindSuspiciosDaemon.Start()
	defer authFindSuspiciosDaemon.Stop()

	//Самомониторинг
	if !debug {
		healthCheckIntervalSec := 60
		h := os.Getenv("HEALTHCHECK_INTERVAL_SEC")
		if h != "" {
			healthCheckIntervalSec, err = strconv.Atoi(h)
			if err != nil || healthCheckIntervalSec <= 0 {
				logger.Logger.Error("invalid HEALTHCHECK_INTERVAL_SEC, using default 5 seconds", zap.Error(err))
			}
		}

		healthCheckDuration := time.Second * time.Duration(healthCheckIntervalSec)

		dbHealthCheck := healthcheck.NewDBHealthCheck(gormDB, healthCheckDuration, logger.Logger, tgSenderHealthCheck, tgChatIDHealthCheck)
		dbHealthCheck.Start()
		defer dbHealthCheck.Stop()

		vpnHealthCheck := healthcheck.NewVpnHealthCheck(healthCheckDuration, logger.Logger, vpnSrv, serverRepo, tgSenderHealthCheck, tgChatIDHealthCheck)
		vpnHealthCheck.Start()
		defer vpnHealthCheck.Stop()

		tgHealthCheck := healthcheck.NewTgHealthCheck(healthCheckDuration, logger.Logger, tgSenderHealthCheck, tgChatIDHealthCheck)
		tgHealthCheck.Start()
		defer tgHealthCheck.Stop()
	}

	r := gin.Default()

	if debug {
		r.SetTrustedProxies(nil)
	} else {
		r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}

	r.Static("/storage", "./storage")

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.NoRoute(middleware.NotFound())

	api := r.Group("/api")
	{
		api.Use(middleware.Recovery(debug, tgSenderHealthCheck, tgChatIDHealthCheck, logger.Logger))
		api.Use(middleware.SetUserIDMiddleware(jwtSrv))

		if len(proofKeys) > 0 {
			log.Println("PROOF ACTIVATED")
			api.Use(middleware.ProofMiddleware(proofKeys, proofBanIfFail, debug))
		}
		v1 := api.Group("/v1")
		{
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
					serverGroup.GET("/:id", serverCtrl.GetServer)
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
				i.GET("/:slug", infoCtrl.GetInfo)
			}
		}
	}

	r.Run(":" + os.Getenv("PORT"))
}
