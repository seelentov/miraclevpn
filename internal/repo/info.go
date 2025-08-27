package repo

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

type InfoRepository struct {
	db *gorm.DB
}

func NewInfoRepository(db *gorm.DB) *InfoRepository {
	return &InfoRepository{
		db: db,
	}
}

func (r *InfoRepository) FindBySlug(slug string) (*models.Info, error) {
	var info models.Info
	if err := r.db.Where("slug = ? AND active = ?", slug, true).First(&info).Error; err != nil {
		return nil, err
	}

	return &info, nil
}

func (r *InfoRepository) FindAll() ([]*models.Info, error) {
	var info []*models.Info
	if err := r.db.Where("active = ?", true).Find(&info).Error; err != nil {
		return nil, err
	}

	return info, nil
}
