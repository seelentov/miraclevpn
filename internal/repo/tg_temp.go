package repo

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

type TgTempRepository struct {
	db *gorm.DB
}

func NewTgTempRepository(db *gorm.DB) *TgTempRepository {
	return &TgTempRepository{
		db,
	}
}

func (r *TgTempRepository) Create(userID int64, message string) error {
	if err := r.db.Save(&models.TgTemp{
		UserID:  userID,
		Message: message,
	}).Error; err != nil {
		return err
	}

	return nil
}
