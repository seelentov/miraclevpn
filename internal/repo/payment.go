package repo

import (
	"miraclevpn/internal/models"
	"time"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	db         *gorm.DB
	expiration time.Duration
}

func NewPaymentRepository(db *gorm.DB, expiration time.Duration) *PaymentRepository {
	return &PaymentRepository{
		db: db,
	}
}

func (r *PaymentRepository) FindByYooKassaID(yooKassaID string) (*models.Payment, error) {
	var p *models.Payment
	if err := r.db.Where("yoo_kassa_id = ?", yooKassaID).First(&p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PaymentRepository) Create(uID string, yooKassaID string, days int) error {
	p := models.Payment{
		UserID:     uID,
		YooKassaID: yooKassaID,
		Days:       days,
	}

	if err := r.db.Save(&p).Error; err != nil {
		return err
	}

	return nil
}

func (r *PaymentRepository) Delete(yooKassaID string) error {
	return r.db.Where("yoo_kassa_id = ?", yooKassaID).Delete(&models.Payment{}).Error
}

func (r *PaymentRepository) DeleteExpired() error {
	return r.db.Where("created_at < ?", time.Now().Add(r.expiration*-1)).Delete(&models.Payment{}).Error
}
