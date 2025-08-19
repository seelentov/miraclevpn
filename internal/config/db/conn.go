package db

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

var migrate = []any{
	&models.User{},
	&models.Server{},
	&models.UserServer{},
	&models.Verifier{},
}

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
