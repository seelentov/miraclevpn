package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	serverdaemon "miraclevpn/internal/daemon/server_daemon"
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

	dbUser := os.Getenv("DB_USER")
	dbHost := os.Getenv("DB_HOST")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbSsl := os.Getenv("DB_SSLMODE")
	dbTZ := os.Getenv("DB_TIMEZONE")
	debug := os.Getenv("DEBUG") == "true"

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewClient(tgTokenHealthCheck, "")

	serverAutoPriorityIntervalStr := os.Getenv("SERVER_AUTO_PRIORITY_INTERVAL_SEC")
	serverAutoPriorityInterval, err := strconv.Atoi(serverAutoPriorityIntervalStr)

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

	if err != nil {
		log.Fatal("failed get SERVER_AUTO_PRIORITY_INTERVAL_SEC: " + err.Error())
	}

	logger, err := logg.NewZapLogger("", 0, debug)
	if err != nil {
		log.Fatal(err)
	}

	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ, "MIIVPN_SERVERDAEMON")
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	serverRepo := repo.NewServerRepository(gormDB)

	ovpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)
	awgSrv := awg.NewClient(awgSSHUser, awgManageScript, awgClientsDir)
	vpnSrv := vpnrouter.NewVpnRouter(ovpnSrv, awgSrv, serverRepo)

	serverAutoPriorityDaemon := serverdaemon.NewServerAutoPriority(time.Second*time.Duration(serverAutoPriorityInterval), logger.Logger, vpnSrv, serverRepo, tgSenderHealthCheck, tgChatIDHealthCheck)
	serverAutoPriorityDaemon.Start()
	defer serverAutoPriorityDaemon.Stop()

	select {}
}
