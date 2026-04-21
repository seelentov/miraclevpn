package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	"miraclevpn/internal/daemon/healthcheck"
	"miraclevpn/internal/repo"
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

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewClient(tgTokenHealthCheck, "")

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

	serverRepo := repo.NewServerRepository(gormDB)

	ovpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)
	awgSrv := awg.NewClient(awgSSHUser, awgManageScript, awgClientsDir)
	vpnSrv := vpnrouter.NewVpnRouter(ovpnSrv, awgSrv, serverRepo)

	healthCheckIntervalSec := 60
	h := os.Getenv("HEALTHCHECK_INTERVAL_SEC")
	if h != "" {
		healthCheckIntervalSec, err = strconv.Atoi(h)
		if err != nil || healthCheckIntervalSec <= 0 {
			logger.Logger.Error("invalid HEALTHCHECK_INTERVAL_SEC, using default 60 seconds", zap.Error(err))
		}
	}

	healthCheckDuration := time.Second * time.Duration(healthCheckIntervalSec)

	dbHealthCheck := healthcheck.NewDBHealthCheck(gormDB, healthCheckDuration, logger.Logger, tgSenderHealthCheck, tgChatIDHealthCheck)
	dbHealthCheck.Start()
	defer dbHealthCheck.Stop()

	vpnHealthCheck := healthcheck.NewVpnHealthCheck(healthCheckDuration, logger.Logger, vpnSrv, serverRepo, tgSenderHealthCheck, tgChatIDHealthCheck)
	vpnHealthCheck.Start()
	defer vpnHealthCheck.Stop()

	tgHealthCheck := healthcheck.NewTgHealthCheck(healthCheckDuration, logger.Logger, tgSenderHealthCheck, tgChatIDHealthCheck, false)
	tgHealthCheck.Start()
	defer tgHealthCheck.Stop()

	select {}
}
