package models

import "time"

type Request struct {
	ID     int64  `gorm:"primaryKey" json:"id"`
	UserID string `json:"user_id"`

	CreatedAt time.Time `json:"created_at"`
	Item    string    `json:"region"`
}
