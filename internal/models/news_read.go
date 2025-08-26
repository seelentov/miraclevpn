package models

import "time"

type NewsRead struct {
	ID     int64  `gorm:"primaryKey"`
	UserID string `gorm:"index"`
	NewsID int64  `gorm:"index"`
	ReadAt time.Time
}
