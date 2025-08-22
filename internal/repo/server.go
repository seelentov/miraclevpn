// Package repo provides data access and storage for the application.
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

func (r *ServerRepository) FindByRegion(region string) ([]*models.Server, error) {
	var s []*models.Server
	if err := r.db.Where("region = ?", region).Find(&s).Error; err != nil {
		return nil, err
	}

	return s, nil
}

func (r *ServerRepository) FindByID(id int64) (*models.Server, error) {
	var s models.Server
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}

	return &s, nil
}

/*
type Region struct {
	Code string
    Name    string
    FlagURL string
}
*/

func (r *ServerRepository) FindAllRegions() ([]*models.Region, error) {
	var regions []*models.Region

	err := r.db.Model(&models.Server{}).
		Select("DISTINCT region, region_name, region_flag_url").
		Where("region IS NOT NULL AND region != ''").
		Order("region").
		Scan(&regions).Error

	if err != nil {
		return nil, err
	}

	return regions, nil
}
