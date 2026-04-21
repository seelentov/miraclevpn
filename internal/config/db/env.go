package db

import (
	"os"

	"gorm.io/gorm"
)

func NewConnFromEnv() (*gorm.DB, error) {
	switch os.Getenv("DB_TYPE") {
	case "sqlite":
		path := os.Getenv("DB_PATH")
		if path == "" {
			path = "miraclevpn.db"
		}
		return NewSQLiteConn(path)
	case "memory":
		return NewInMemoryConn()
	default:
		return NewPostgreConn(
			os.Getenv("DB_HOST"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_NAME"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_SSLMODE"),
			os.Getenv("DB_TIMEZONE"),
			"MIIVPN",
		)
	}
}
