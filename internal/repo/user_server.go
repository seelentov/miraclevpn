package repo

import (
	"miraclevpn/internal/models"
	"time"

	"gorm.io/gorm"
)

type UserServerRepository struct {
	db *gorm.DB
}

func NewUserServerRepository(db *gorm.DB) *UserServerRepository {
	return &UserServerRepository{
		db,
	}
}

func (r *UserServerRepository) FindByUserIDServerID(userID string, serverID int64) (*models.UserServer, error) {
	var userServer models.UserServer
	if err := r.db.Where("user_id = ? AND server_id = ?", userID, serverID).First(&userServer).Error; err != nil {
		return nil, err
	}

	userServer.ConfigFile = ""

	return &userServer, nil
}

func (r *UserServerRepository) CreateOrUpdate(userID string, serverID int64, config string, configFile string) error {
	userServer := models.UserServer{
		UserID:     userID,
		ServerID:   serverID,
		Config:     config,
		ConfigFile: configFile,
	}
	if err := r.db.Save(&userServer).Error; err != nil {
		return err
	}
	return nil
}

func (r *UserServerRepository) FindExpired(expiration time.Duration) ([]*models.UserServer, error) {
	var us []*models.UserServer
	if err := r.db.Where("updated_at < ?", time.Now().Add(expiration*-1)).Find(&us).Error; err != nil {
		return nil, err
	}

	return us, nil
}

func (r *UserServerRepository) FindExpiredByUser() ([]*models.UserServer, error) {
	var us []*models.UserServer
	if err := r.db.Where("updated_at < ?", time.Now()).Find(&us).Error; err != nil {
		return nil, err
	}

	return us, nil
}
