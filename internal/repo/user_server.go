package repo

import (
	"miraclevpn/internal/models"

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

func (r *UserServerRepository) FindByUserIDServerID(userID int64, serverID int64) (*models.UserServer, error) {
	var userServer models.UserServer
	if err := r.db.Where("user_id = ? AND server_id = ?", userID, serverID).First(&userServer).Error; err != nil {
		return nil, err
	}
	return &userServer, nil
}

func (r *UserServerRepository) CreateOrUpdate(userID int64, serverID int64, config string) error {
	userServer := models.UserServer{
		UserID:   userID,
		ServerID: serverID,
		Config:   config,
	}
	if err := r.db.Save(&userServer).Error; err != nil {
		return err
	}
	return nil
}
