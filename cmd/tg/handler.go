package main

import (
	"log"
	"math"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	tgcontroller "miraclevpn/internal/controller/tg"
	"miraclevpn/internal/controller/tg/controller"
	"miraclevpn/internal/controller/tg/middleware"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/auth"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/servers"
	"miraclevpn/pkg/ovpn"
	"miraclevpn/pkg/tg"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
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
	debug := os.Getenv("DEBUG") == "true"

	sshUser := os.Getenv("SSH_USER")
	sshStatusPath := os.Getenv("SSH_STATUS_PATH")
	sshCreateUserFile := os.Getenv("SSH_CREATE_USER_FILE")
	sshRevokeUserFile := os.Getenv("SSH_REVOKE_USER_FILE")
	sshConfigsDir := os.Getenv("SSH_CONFIGS_DIR")

	tgToken := os.Getenv("TG_HANDLER_TOKEN")

	freeTrial, err := strconv.Atoi(os.Getenv("FREE_TRIAL_SEC"))
	if err != nil {
		log.Fatal("failed get FREE_TRIAL_SEC: " + err.Error())
	}

	jwtSecretAuth := os.Getenv("JWT_SECRET_AUTH")
	jwtDuration := math.MaxInt32
	if os.Getenv("JWT_DURATION_MIN") != "" && os.Getenv("JWT_DURATION_MIN") != "0" {
		jwtDuration, _ = strconv.Atoi(os.Getenv("JWT_DURATION_MIN"))
	}

	logger, err := logg.NewZapLogger("", 0, debug)
	if err != nil {
		log.Fatal(err)
	}

	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ, "MIIVPN_TGHANDLER")
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
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

	userRepo := repo.NewUserRepository(gormDB, argonSrv, time.Duration(freeTrial)*time.Second)
	userServerRepo := repo.NewUserServerRepository(gormDB)
	serverRepo := repo.NewServerRepository(gormDB)
	authDataRepo := repo.NewAuthDataRepository(gormDB)

	vpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)

	authSrv := auth.NewAuthService(userRepo, authDataRepo, jwtSrv, time.Duration(jwtDuration)*time.Minute, logger.Logger)
	serversSrv := servers.NewServersService(userServerRepo, serverRepo, userRepo, vpnSrv, logger.Logger)

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewClient(tgTokenHealthCheck, "")

	paymentURL := os.Getenv("PAYMENT_URL")
	lkURL := os.Getenv("LK_URL")
	indexCtrl := controller.NewIndexTGController(paymentURL, lkURL)
	authCtrl := controller.NewAuthTGController()
	connectCtrl := controller.NewConnectTGController(serversSrv)

	r, err := tgcontroller.NewRouter(tgToken)
	if err != nil {
		panic(err)
	}

	r.Use404(middleware.NotFoundHandler())
	r.UseRecover(middleware.RecoverrHandler(debug, tgSenderHealthCheck, tgChatIDHealthCheck, logger.Logger))
	r.Use(middleware.AuthMiddlewareTg(authSrv, userRepo))

	r.UseHandler("/start", indexCtrl.Index)
	r.UseHandler("/menu", indexCtrl.Index)
	r.UseHandler("/get_key", authCtrl.GetToken)
	r.UseHandler("/gift", indexCtrl.FreeForReview)

	r.UseHandler("/servers", connectCtrl.Index)
	r.UseHandler("/connect", connectCtrl.GetConfig)
	r.UseHandler("/servers_all", connectCtrl.GetAll)
	r.UseHandler("/stats", connectCtrl.GetStats)

	logger.Logger.Info("Starting...")
	r.Start()
}
