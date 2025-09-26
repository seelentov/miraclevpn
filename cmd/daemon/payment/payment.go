package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	paymentdaemon "miraclevpn/internal/daemon/payment_daemon"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/payment"
	"miraclevpn/pkg/tg"
	"miraclevpn/pkg/yookassa"
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

	paymentAutoIntervalStr := os.Getenv("PAYMENT_AUTO_INTERVAL_SEC")
	paymentAutoInterval, err := strconv.Atoi(paymentAutoIntervalStr)

	if err != nil {
		log.Fatal("failed get PAYMENT_AUTO_INTERVAL_SEC: " + err.Error())
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

	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ, "MIIVPN_PAYMENTDAEMON")
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	payRepo := repo.NewPaymentRepository(gormDB, time.Second*time.Duration(paymentExpiration))
	payPlRepo := repo.NewPaymentPlanRepository(gormDB)
	userRepo := repo.NewUserRepository(gormDB, nil, 0)

	paymentClient := yookassa.NewClient(os.Getenv("PAYMENT_SHOP_ID"), os.Getenv("PAYMENT_SECRET"), os.Getenv("PAYMENT_RETURN_URL"))

	paySrv := payment.NewAutoPaymentService(paymentClient, payRepo, payPlRepo, userRepo, logger.Logger)

	paymentClearDaemon := paymentdaemon.NewPaymentRemoveExpired(time.Second*time.Duration(paymentRemoveExpiredInterval), logger.Logger, payRepo, tgSenderHealthCheck, tgChatIDHealthCheck)
	paymentClearDaemon.Start()
	defer paymentClearDaemon.Stop()

	paymentAutoDaemon := paymentdaemon.NewAutoPaymentDaemon(time.Second*time.Duration(paymentAutoInterval), logger.Logger, paySrv, tgSenderHealthCheck, tgChatIDHealthCheck)
	paymentAutoDaemon.Start()
	defer paymentAutoDaemon.Stop()

	select {}
}
