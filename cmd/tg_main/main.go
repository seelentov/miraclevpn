package main

import (
	"log"
	"os"

	"miraclevpn/internal/config/db"
	"miraclevpn/internal/config/logg"
	"miraclevpn/internal/daemon/tg"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	logger, err := logg.NewZapLogger("logs", 1, true)
	if err != nil {
		log.Fatal(err)
	}

	// Получение параметров из окружения
	botToken := os.Getenv("TG_TOKEN")
	jwtSecret := os.Getenv("JWT_SECRET")

	dbUser := os.Getenv("DB_USER")
	dbHost := os.Getenv("DB_HOST")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbSsl := os.Getenv("DB_SSLMODE")
	dbTZ := os.Getenv("DB_TIMEZONE")

	// // Подключение к БД
	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ)
	if err != nil {
		logger.Logger.Fatal("failed to connect to db", zap.Error(err))
	}

	// Криптосервисы
	argonParams := &crypt.Argon2idParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
	argonSrv := crypt.NewArgonService(argonParams, logger.Logger)

	userRepo := repo.NewUserRepository(gormDB, argonSrv)
	jwtSrv := crypt.NewJwtService(jwtSecret, nil)

	daemon := tg.NewTgDaemon(botToken, jwtSrv, userRepo)
	daemon.Start()

	log.Println("Telegram daemon started. Press Ctrl+C to exit.")

	select {}
}
