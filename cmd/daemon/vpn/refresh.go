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

	debug := os.Getenv("DEBUG") == "true"

	sshUser := os.Getenv("OVPN_SSH_USER")
	sshStatusPath := os.Getenv("OVPN_STATUS_PATH")
	sshCreateUserFile := os.Getenv("OVPN_CREATE_USER_FILE")
	sshRevokeUserFile := os.Getenv("OVPN_REVOKE_USER_FILE")
	sshConfigsDir := os.Getenv("OVPN_CONFIGS_DIR")

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

	gormDB, err := db.NewConnFromEnv()
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	vpnRemoveExpiredIntervalStr := os.Getenv("VPN_REMOVE_EXPIRED_INTERVAL_SEC")
	vpnRemoveExpiredInterval, err := strconv.Atoi(vpnRemoveExpiredIntervalStr)
	if err != nil {
		log.Fatal("failed get VPN_REMOVE_EXPIRED_INTERVAL_SEC: " + err.Error())
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

	vpnRemoveExpiredDaemon := vpndaemon.NewVpnRemoveExpiredDaemon(time.Second*time.Duration(vpnRemoveExpiredInterval), logger.Logger, serversSrv, tgSenderHealthCheck, tgChatIDHealthCheck)
	vpnRemoveExpiredDaemon.Start()
	defer vpnRemoveExpiredDaemon.Stop()

	select {}
}
