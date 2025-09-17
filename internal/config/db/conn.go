// Package db provides database connection utilities for the application.
package db

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

var migrate = []any{
	&models.User{},
	&models.Server{},
	&models.UserServer{},
	&models.Info{},
	&models.KeyValue{},
	&models.News{},
	&models.NewsRead{},
	&models.PaymentPlan{},
	&models.Request{},
	&models.AuthData{},
	&models.Payment{},
}

func NewConn(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(migrate...); err != nil {
		return nil, err
	}

	if err := seed(db); err != nil {
		return nil, err
	}

	return db, nil
}

func seed(db *gorm.DB) (err error) {
	return
}
