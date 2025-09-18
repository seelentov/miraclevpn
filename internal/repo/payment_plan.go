package repo

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

type PaymentPlanRepository struct {
	db *gorm.DB
}

func NewPaymentPlanRepository(db *gorm.DB) *PaymentPlanRepository {
	return &PaymentPlanRepository{
		db: db,
	}
}

func (r *PaymentPlanRepository) FindAll() ([]*models.PaymentPlan, error) {
	var p []*models.PaymentPlan
	if err := r.db.Where("active = ?", true).Order("price").Find(&p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PaymentPlanRepository) FindByID(planID int64) (*models.PaymentPlan, error) {
	var p *models.PaymentPlan
	if err := r.db.Where("id = ? and active = ?", planID, true).First(&p).Error; err != nil {
		return nil, err
	}
	return p, nil
}
