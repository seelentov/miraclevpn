package db

import (
	"gorm.io/gorm"
)

var migrate = []any{}

func NewConn(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(migrate...); err != nil {
		return nil, err
	}

	return db, nil
}
