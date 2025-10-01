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

	userServer.UpdatedAt = time.Now()
	if err := r.db.Save(&userServer).Error; err != nil {
		return nil, err
	}

	userServer.ConfigFile = ""
	userServer.ConfigFileExpired = nil

	return &userServer, nil
}

func (r *UserServerRepository) CreateOrUpdate(userID string, serverID int64, config string, configFile string, configFileExpired *string) error {
	userServer := models.UserServer{
		UserID:            userID,
		ServerID:          serverID,
		Config:            config,
		ConfigFile:        configFile,
		ConfigFileExpired: configFileExpired,
		UpdatedAt:         time.Now(),
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

	expiredUserIDs := r.db.
		Table("users").
		Select("id").
		Where("expired_at < ?", time.Now())

	result := r.db.
		Table("user_servers").
		Where("user_id IN (?)", expiredUserIDs).Find(&us)

	return us, result.Error
}

func (r *UserServerRepository) Delete(uss []*models.UserServer) error {
	if len(uss) == 0 {
		return nil
	}

	db := r.db
	for i, us := range uss {
		if i == 0 {
			db = db.Where("(user_id = ? AND server_id = ?)", us.UserID, us.ServerID)
		} else {
			db = db.Or("(user_id = ? AND server_id = ?)", us.UserID, us.ServerID)
		}
	}

	return db.Delete(&models.UserServer{}).Error
}

func (r *UserServerRepository) FindByConfigFile(configFile string, orExpired bool) (*models.UserServer, error) {
	var us *models.UserServer
	if err := r.db.Where("config_file = ? OR config_file_expired = ?", configFile, configFile).First(&us).Error; err != nil {
		return nil, err
	}
	return us, nil
}

func (r *UserServerRepository) UpdateExpirationByConfigFile(configFile string, date time.Time) error {
	us, err := r.FindByConfigFile(configFile, false)
	if err != nil {
		return err
	}

	us.UpdatedAt = date

	return r.db.Save(us).Error
}
