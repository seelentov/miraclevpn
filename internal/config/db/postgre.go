package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgreConn(user, password, dbname, port, sslmode, timeZone string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s", user, password, dbname, port, sslmode, timeZone)
	return NewConn(postgres.Open(dsn))
}
