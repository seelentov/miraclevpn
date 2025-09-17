package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	paymentdaemon "miraclevpn/internal/daemon/payment_daemon"
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
	debug := os.Getenv("DEBUG") == "true"

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewTgClient(tgTokenHealthCheck, "")

	paymentRemoveExpiredIntervalStr := os.Getenv("PAYMENT_REMOVE_EXPIRED_INTERVAL_SEC")
	paymentRemoveExpiredInterval, err := strconv.Atoi(paymentRemoveExpiredIntervalStr)

	if err != nil {
		log.Fatal("failed get PAYMENT_REMOVE_EXPIRED_INTERVAL_SEC: " + err.Error())
	}

	paymentExpirationStr := os.Getenv("PAYMENT_EXPIRATION_SEC")
	paymentExpiration, err := strconv.Atoi(paymentExpirationStr)

	if err != nil {
		log.Fatal("failed get PAYMENT_EXPIRATION_SEC: " + err.Error())
	}

	logger, err := logg.NewZapLogger("", 0, debug)
	if err != nil {
		log.Fatal(err)
	}

	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ)
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	payRepo := repo.NewPaymentRepository(gormDB, time.Second*time.Duration(paymentExpiration))

	paymentClearDaemon := paymentdaemon.NewPaymentRemoveExpired(time.Second*time.Duration(paymentRemoveExpiredInterval), logger.Logger, payRepo, tgSenderHealthCheck, tgChatIDHealthCheck)
	paymentClearDaemon.Start()
	defer paymentClearDaemon.Stop()

	select {}
}
