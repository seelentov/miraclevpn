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
	vpnrouter "miraclevpn/internal/services/vpn"
	"miraclevpn/pkg/awg"
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

	debug := os.Getenv("DEBUG") == "true"

	sshUser := os.Getenv("OVPN_SSH_USER")
	sshStatusPath := os.Getenv("OVPN_STATUS_PATH")
	sshCreateUserFile := os.Getenv("OVPN_CREATE_USER_FILE")
	sshRevokeUserFile := os.Getenv("OVPN_REVOKE_USER_FILE")
	sshConfigsDir := os.Getenv("OVPN_CONFIGS_DIR")

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

	gormDB, err := db.NewConnFromEnv()
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

	ovpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)
	awgSrv := awg.NewClient(awgSSHUser, awgManageScript, awgClientsDir)
	vpnSrv := vpnrouter.NewVpnRouter(ovpnSrv, awgSrv, serverRepo)

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
	r.UseHandler("/quick_connect", connectCtrl.QuickConnect)
	r.UseHandler("/connect", connectCtrl.Connect)

	logger.Logger.Info("Starting old...")

	r.Start()
}
