package repo

import (
	"miraclevpn/internal/models"
	"time"

	"gorm.io/gorm"
)

type AuthDataRepository struct {
	db *gorm.DB
}

func NewAuthDataRepository(db *gorm.DB) *AuthDataRepository {
	return &AuthDataRepository{
		db: db,
	}
}

func (r *AuthDataRepository) Add(uid string, data map[string]interface{}) error {
	if err := r.db.Save(&models.AuthData{
		Data: data,
		UID:  uid,
		Date: time.Now(),
	}).Error; err != nil {
		return err
	}

	return nil
}
