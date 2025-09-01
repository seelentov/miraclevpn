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

func (r *AuthDataRepository) FindSuspicios() ([]*models.AuthData, error) {
	var results []*models.AuthData

	subQuery := r.db.Model(&models.AuthData{}).
		Select("data->>'ip' as ip, COUNT(DISTINCT uid) as uid_count").
		Where("data ? 'ip'").
		Group("data->>'ip'").
		Having("COUNT(DISTINCT uid) > 1")

	err := r.db.
		Joins("INNER JOIN (?) AS dup_ips ON auth_data.data->>'ip' = dup_ips.ip", subQuery).
		Find(&results).Error

	return results, err
}
