// Package repo provides data access and storage for the application.
package repo

import (
	"errors"
	"miraclevpn/internal/models"
	"time"

	"gorm.io/gorm"
)

var (
	ErrReqAlreadyExist = errors.New("request already exist")
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
	if err := r.db.Where("active = ? AND preview = ?", true, false).Order("priority DESC").Find(&s).Error; err != nil {
		return nil, err
	}
	return s, nil
}

func (r *ServerRepository) FindBest() ([]*models.Server, error) {
	var s []*models.Server

	subquery := r.db.Model(&models.Server{}).
		Select("region, MAX(priority) as max_priority").
		Where("active = ? AND preview = ?", true, false).
		Group("region")

	err := r.db.
		Joins("INNER JOIN (?) as mp ON servers.region = mp.region AND servers.priority = mp.max_priority", subquery).
		Where("servers.active = ?", true).
		Order("servers.region").
		Find(&s).Error

	if err != nil {
		return nil, err
	}

	return s, nil
}

func (r *ServerRepository) FindSuperBest() (*models.Server, error) {
	var s models.Server
	if err := r.db.Where("active = ? AND preview = ?", true, false).Order("priority DESC").First(&s).Error; err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *ServerRepository) FindByRegion(region string) ([]*models.Server, error) {
	var s []*models.Server
	if err := r.db.Where("region = ? AND active = ? AND preview = ?", region, true, false).Order("priority DESC").Find(&s).Error; err != nil {
		return nil, err
	}

	return s, nil
}

func (r *ServerRepository) FindByID(id int64) (*models.Server, error) {
	var s models.Server
	if err := r.db.Where("id = ? AND active = ?", id, true).First(&s).Error; err != nil {
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
		Where("region IS NOT NULL AND region != '' AND active = ? AND preview = ?", true, false).
		Order("region").
		Scan(&regions).Error

	if err != nil {
		return nil, err
	}

	return regions, nil
}

func (r *ServerRepository) FindPreview() ([]*models.Server, error) {
	var s []*models.Server
	if err := r.db.Where("active = ? AND preview = ?", true, true).Find(&s).Error; err != nil {
		return nil, err
	}

	return s, nil
}

func (r *ServerRepository) RequestExist(item string, userID string) (bool, error) {
	var re models.Request
	if err := r.db.Where("user_id = ? AND item = ?", userID, item).First(&re).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (r *ServerRepository) SendRequest(item string, userID string) error {
	exist, err := r.RequestExist(item, userID)
	if err != nil {
		return err
	}
	if exist {
		return ErrReqAlreadyExist
	}

	if err := r.db.Save(&models.Request{
		UserID:    userID,
		Item:      item,
		CreatedAt: time.Now(),
	}).Error; err != nil {
		return err
	}
	return nil
}

func (r *ServerRepository) FindByHost(host string) (*models.Server, error) {
	var s models.Server
	if err := r.db.Where("host = ?", host).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServerRepository) UpdatePriority(id int64, priority int) error {
	s, err := r.FindByID(id)
	if err != nil {
		return err
	}

	s.Priority = priority

	return r.db.Save(s).Error
}
