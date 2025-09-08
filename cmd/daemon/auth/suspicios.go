package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	authdaemon "miraclevpn/internal/daemon/auth_daemon"
	"miraclevpn/internal/repo"
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

	logger, err := logg.NewZapLogger(logDir, logRetain, debug)
	if err != nil {
		log.Fatal(err)
	}

	authFindSuspiciousIntervalStr := os.Getenv("AUTH_FIND_SUSPICIOUS_INTERVAL_SEC")
	authFindSuspiciousInterval, err := strconv.Atoi(authFindSuspiciousIntervalStr)

	if err != nil {
		log.Fatal("AUTH_FIND_SUSPICIOUS_INTERVAL_SEC empty: ", err)
	}

	// Подключение к БД PostgreSQL
	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ)
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	authDataRepo := repo.NewAuthDataRepository(gormDB)

	authFindSuspiciosDaemon := authdaemon.NewAuthFindSuspicious(time.Second*time.Duration(authFindSuspiciousInterval), logger.Logger, authDataRepo, tgSenderHealthCheck, tgChatIDHealthCheck)
	authFindSuspiciosDaemon.Start()
	defer authFindSuspiciosDaemon.Stop()

}
