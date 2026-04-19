package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	vpndaemon "miraclevpn/internal/daemon/vpn_daemon"
	"miraclevpn/internal/repo"
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

	logger, err := logg.NewZapLogger("", 0, debug)
	if err != nil {
		log.Fatal(err)
	}

	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ, "MIIVPN_VPNDAEMON")
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	vpnKickHighloadStr := os.Getenv("VPN_KICK_HIGHLOAD_INTERVAL_SEC")
	vpnKickHighload, err := strconv.Atoi(vpnKickHighloadStr)
	if err != nil {
		log.Fatal("failed get VPN_KICK_HIGHLOAD_INTERVAL_SEC: " + err.Error())
	}

	vpnKickHighloadBytesStr := os.Getenv("VPN_KICK_HIGHLOAD_BYTES")
	vpnKickHighloadBytes, err := strconv.ParseInt(vpnKickHighloadBytesStr, 10, 64)
	if err != nil {
		log.Fatal("failed get VPN_KICK_HIGHLOAD_BYTES: " + err.Error())
	}

	freeTrial, err := strconv.Atoi(os.Getenv("FREE_TRIAL_SEC"))
	if err != nil {
		log.Fatal("failed get FREE_TRIAL_SEC: " + err.Error())
	}

	argonParams := &crypt.Argon2idParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}

	argonSrv := crypt.NewArgonService(argonParams, logger.Logger)

	userRepo := repo.NewUserRepository(gormDB, argonSrv, time.Duration(freeTrial)*time.Second)
	serverRepo := repo.NewServerRepository(gormDB)
	userServerRepo := repo.NewUserServerRepository(gormDB)

	ovpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)
	awgSrv := awg.NewClient(awgSSHUser, awgManageScript, awgClientsDir)
	vpnSrv := vpnrouter.NewVpnRouter(ovpnSrv, awgSrv, serverRepo)

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewClient(tgTokenHealthCheck, "")

	serversSrv := servers.NewServersService(userServerRepo, serverRepo, userRepo, vpnSrv, logger.Logger)

	vpnKickHighloadDaemon := vpndaemon.NewKickHighloadDaemon(time.Second*time.Duration(vpnKickHighload), logger.Logger, serversSrv, vpnSrv, tgSenderHealthCheck, tgChatIDHealthCheck, time.Second*10, vpnKickHighloadBytes, vpnKickHighload)
	vpnKickHighloadDaemon.Start()
	defer vpnKickHighloadDaemon.Stop()

	select {}
}
