package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	"miraclevpn/internal/daemon/healthcheck"
	"miraclevpn/internal/repo"
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
	logDir := os.Getenv("LOG_DIR")
	logRetain, _ := strconv.Atoi(os.Getenv("LOG_RETAIN"))
	debug := os.Getenv("DEBUG") == "true"

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewTgClient(tgTokenHealthCheck, "")

	sshUser := os.Getenv("SSH_USER")
	sshStatusPath := os.Getenv("SSH_STATUS_PATH")
	sshCreateUserFile := os.Getenv("SSH_CREATE_USER_FILE")
	sshRevokeUserFile := os.Getenv("SSH_REVOKE_USER_FILE")
	sshConfigsDir := os.Getenv("SSH_CONFIGS_DIR")

	logger, err := logg.NewZapLogger(logDir, logRetain, debug)
	if err != nil {
		log.Fatal(err)
	}

	// Подключение к БД PostgreSQL
	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ)
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	vpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)

	serverRepo := repo.NewServerRepository(gormDB)

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
