package repo

import (
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) FindActive() ([]*models.Review, error) {
	var reviews []*models.Review
	if err := r.db.Where("active = ?", true).Order("sort_order ASC, id ASC").Find(&reviews).Error; err != nil {
		return nil, err
	}
	return reviews, nil
}
