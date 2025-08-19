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

func (r *TgTempRepository) FindByUserID(userID int64) (*models.TgTemp, error) {
	var tgTemp models.TgTemp
	if err := r.db.Where("user_id = ?", userID).Find(&tgTemp).Error; err != nil {
		return nil, err
	}
	return &tgTemp, nil
}

func (r *TgTempRepository) DeleteByUserID(userID int64) error {
	if err := r.db.Where("user_id = ?", userID).Delete(&models.TgTemp{}).Error; err != nil {
		return err
	}

	return nil
}
