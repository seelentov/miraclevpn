package repo

import (
	"errors"
	"math/rand"
	"miraclevpn/internal/models"
	"time"

	"gorm.io/gorm"
)

type VerifierRepository struct {
	db *gorm.DB
}

func NewVerifierRepository(db *gorm.DB) *VerifierRepository {
	return &VerifierRepository{
		db,
	}
}

func (r *VerifierRepository) Verify(userID int64, code int32) (bool, error) {
	var verifier models.Verifier
	if err := r.db.Where("user_id = ? AND code = ? AND expired_at > ?", userID, code, time.Now()).First(&verifier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *VerifierRepository) Create(userID int64, expiredAt time.Time) (int32, error) {

	code := generateCode()

	if err := r.db.Create(&models.Verifier{
		UserID:    userID,
		Code:      code,
		ExpiredAt: expiredAt,
	}).Error; err != nil {
		return 0, err
	}

	return code, nil
}

func (r *VerifierRepository) DeleteByUserID(userID int64) error {
	if err := r.db.Where("user_id = ?", userID).Delete(&models.Verifier{}).Error; err != nil {
		return err
	}
	return nil
}

func generateCode() int32 {
	return rand.Int31n(899999) + 100000
}
