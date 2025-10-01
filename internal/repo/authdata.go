package repo

import (
	"errors"
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

func (r *AuthDataRepository) FindLatest(userID string) (*models.AuthData, error) {
	var m models.AuthData
	err := r.db.Where("uid = ?", userID).Order("date DESC").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &m, nil
}
