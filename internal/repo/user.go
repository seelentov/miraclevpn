package repo

import (
	"miraclevpn/internal/models"
	cryptt "miraclevpn/internal/services/crypt"
	"time"

	"gorm.io/gorm"
)

type UserRepository struct {
	db        *gorm.DB
	crypt     cryptt.CryptService
	freeTrial time.Duration
}

func NewUserRepository(db *gorm.DB, crypt cryptt.CryptService, freeTrial time.Duration) *UserRepository {
	return &UserRepository{db, crypt, freeTrial}
}

func (r *UserRepository) FindByID(userID int64) (*models.User, error) {
	var u models.User

	if err := r.db.First(&u, userID).Error; err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepository) Create(uID int64) (*models.User, error) {
	u := models.User{
		ID:        uID,
		Trial:     true,
		ExpiredAt: time.Now().Add(r.freeTrial),
	}

	if err := r.db.Save(&u).Error; err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepository) AddSubDays(userID int64, days int) error {
	var u models.User
	if err := r.db.Find(&u, userID).Error; err != nil {
		return err
	}

	u.ExpiredAt = u.ExpiredAt.Add(time.Duration(days) * time.Hour * 24)
	u.Trial = false

	if err := r.db.Save(&u).Error; err != nil {
		return err
	}

	return nil
}
