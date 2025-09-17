package models

import "time"

type Payment struct {
	UserID     string `gorm:"primaryKey" json:"user_id"`
	YooKassaID string `gorm:"primaryKey" json:"yoo_kassa_id"`

	Days int

	CreatedAt time.Time
}
