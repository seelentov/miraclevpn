package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewSQLiteConn(path string) (*gorm.DB, error) {
	return NewConn(sqlite.Open(path))
}
