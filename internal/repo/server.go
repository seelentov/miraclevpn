package repo

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

type ServerRepository struct {
	db *gorm.DB
}

func NewServerRepository(db *gorm.DB) *ServerRepository {
	return &ServerRepository{
		db,
	}
}

func (r *ServerRepository) FindAll() ([]*models.Server, error) {
	var s []*models.Server
	if err := r.db.Find(&s).Error; err != nil {
		return nil, err
	}
	return s, nil
}

func (r *ServerRepository) FindByRegionAll(region string) ([]*models.Server, error) {
	var s []*models.Server
	if err := r.db.Find(&s).Where("region = ?", region).Error; err != nil {
		return nil, err
	}

	return s, nil
}
