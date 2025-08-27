package models

import "time"

type NewsRead struct {
	ID     int64     `gorm:"primaryKey" json:"id"`
	UserID string    `gorm:"index" json:"user_id"`
	NewsID int64     `gorm:"index" json:"news_id"`
	ReadAt time.Time `json:"read_at"`
}
