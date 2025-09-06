package repo

import (
	"miraclevpn/internal/models"
	"time"

	"gorm.io/gorm"
)

type NewsRepository struct {
	db *gorm.DB
}

func NewNewsRepository(db *gorm.DB) *NewsRepository {
	return &NewsRepository{
		db: db,
	}
}

func (r *NewsRepository) FindUnread(userID string) ([]*models.News, error) {
	var news []*models.News
	if err := r.db.Where("id NOT IN (SELECT news_id FROM news_reads WHERE user_id = ?) AND active = ?", userID, true).
		Find(&news).Error; err != nil {
		return nil, err
	}

	if len(news) > 0 {
		now := time.Now()
		newsReads := make([]*models.NewsRead, len(news))

		for i, n := range news {
			if n.Repeat {
				continue
			}
			newsReads[i] = &models.NewsRead{
				UserID: userID,
				NewsID: n.ID,
				ReadAt: now,
			}
		}

		if err := r.db.Create(&newsReads).Error; err != nil {
			return nil, err
		}
	}

	return news, nil
}
