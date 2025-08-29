package models

import "time"

type Requests struct {
	ID     int64  `gorm:"primaryKey" json:"id"`
	UserID string `json:"user_id"`

	CreatedAt time.Time `json:"created_at"`
	Region    string    `json:"region"`
}
