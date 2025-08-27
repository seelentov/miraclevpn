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
	&models.Requests{},
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
	err = keyValuesSeed(db)
	return
}

func keyValuesSeed(db *gorm.DB) error {
	db.Save(&models.KeyValue{
		Key:   "tech_work",
		Value: "false",
	})
	db.Save(&models.KeyValue{
		Key:   "tech_work_text",
		Value: "В данный момент на сайте технические работы",
	})
	return nil
}
