package main

import (
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	vpndaemon "miraclevpn/internal/daemon/vpn_daemon"
	"miraclevpn/internal/repo"
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

	vpnConfigExpirationStt := os.Getenv("VPN_CONFIG_DIRATION_SEC")

	vpnConfigExpiration, err := strconv.Atoi(vpnConfigExpirationStt)
	if err != nil {
		log.Fatal("failed get VPN_CONFIG_DIRATION_SEC: " + err.Error())
	}

	logger, err := logg.NewZapLogger("", 0, debug)
	if err != nil {
		log.Fatal(err)
	}

	// Подключение к БД PostgreSQL
	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ, "MIIVPN_VPNDAEMON")
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	vpnRefreshConfigIntervalStr := os.Getenv("VPN_REFRESH_INTERVAL_SEC")

	vpnRefreshConfigInterval, err := strconv.Atoi(vpnRefreshConfigIntervalStr)
	if err != nil {
		log.Fatal("failed get VPN_REFRESH_INTERVAL_SEC: " + err.Error())
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

	vpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)

	userRepo := repo.NewUserRepository(gormDB, argonSrv, time.Duration(freeTrial)*time.Second)
	serverRepo := repo.NewServerRepository(gormDB)
	userServerRepo := repo.NewUserServerRepository(gormDB)

	tgTokenHealthCheck := os.Getenv("TG_HEALTHCHECK_TOKEN")
	tgChatIDHealthCheck := os.Getenv("TG_HEALTHCHECK_CHAT_ID")
	tgSenderHealthCheck := tg.NewClient(tgTokenHealthCheck, "")

	serversSrv := servers.NewServersService(userServerRepo, serverRepo, userRepo, vpnSrv, logger.Logger)

	vpnRefreshDaemon := vpndaemon.NewVpnRefreshDaemon(time.Second*time.Duration(vpnRefreshConfigInterval), logger.Logger, serversSrv, tgSenderHealthCheck, tgChatIDHealthCheck, time.Second*time.Duration(vpnConfigExpiration))
	vpnRefreshDaemon.Start()
	defer vpnRefreshDaemon.Stop()

	vpnRemoveExpiredDaemon := vpndaemon.NewVpnRemoveExpiredDaemon(time.Second*time.Duration(vpnRemoveExpiredInterval), logger.Logger, serversSrv, tgSenderHealthCheck, tgChatIDHealthCheck)
	vpnRemoveExpiredDaemon.Start()
	defer vpnRemoveExpiredDaemon.Stop()

	select {}
}
