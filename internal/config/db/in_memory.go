package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewInMemoryConn() (*gorm.DB, error) {
	return NewConn(sqlite.Open(":memory:"))
}
