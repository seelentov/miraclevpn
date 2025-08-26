package repo

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

type NewsRepository struct {
	db *gorm.DB
}

func NewNewsRepository(db *gorm.DB) *NewsRepository {
	return &NewsRepository{
		db: db,
	}
}

func (r *NewsRepository) FindUnreaded(userID string) ([]*models.News, error) {
	var news []*models.News
	if err := r.db.Where("readers NOT LIKE ?", "%"+userID+"%").Find(&news).Error; err != nil {
		return nil, err
	}

	for _, n := range news {
		n.Readers += userID + ","
	}

	if err := r.db.Save(&news).Error; err != nil {
		return nil, err
	}

	return news, nil
}
