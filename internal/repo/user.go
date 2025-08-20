package repo

import (
	"miraclevpn/internal/models"
	cryptt "miraclevpn/internal/services/crypt"
	"time"

	"gorm.io/gorm"
)

type UserRepository struct {
	db    *gorm.DB
	crypt cryptt.CryptService
}

func NewUserRepository(db *gorm.DB, crypt cryptt.CryptService) *UserRepository {
	return &UserRepository{db, crypt}
}

func (r *UserRepository) FindByID(userID int64) (*models.User, error) {
	var u models.User

	if err := r.db.Find(&u, userID).Error; err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	var u models.User

	if err := r.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepository) Create(username string, password string) (*models.User, error) {
	hashedPassword, err := r.crypt.GenerateHash(password)
	if err != nil {
		return nil, err
	}

	u := models.User{
		Username: username,
		Password: hashedPassword,
		Active:   false,
	}

	if err := r.db.Save(&u).Error; err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepository) SetPassword(userID int64, newPassword string) error {
	var u models.User
	if err := r.db.Find(&u, userID).Error; err != nil {
		return err
	}

	hashedPassword, err := r.crypt.GenerateHash(newPassword)
	if err != nil {
		return err
	}

	u.Password = hashedPassword

	if err := r.db.Save(&u).Error; err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) SetTGChatID(userID, chatID int64) error {
	var u models.User
	if err := r.db.Find(&u, userID).Error; err != nil {
		return err
	}

	u.TGChat = &chatID

	if err := r.db.Save(&u).Error; err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) Activate(userID int64) error {
	var u models.User
	if err := r.db.Find(&u, userID).Error; err != nil {
		return err
	}

	u.Active = true

	if err := r.db.Save(&u).Error; err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) AddSubDays(userID int64, days int) error {
	var u models.User
	if err := r.db.Find(&u, userID).Error; err != nil {
		return err
	}

	u.ExpiredAt = u.ExpiredAt.Add(time.Duration(days) * time.Hour * 24)

	if err := r.db.Save(&u).Error; err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) CheckPassword(userID int64, password string) bool {
	u, err := r.FindByID(userID)
	if err != nil {
		return false
	}

	ok, err := r.crypt.ComparePasswordAndHash(password, u.Password)
	if err != nil {
		return false
	}
	return ok
}
