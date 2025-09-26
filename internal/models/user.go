package models

import "time"

type User struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	ExpiredAt time.Time `json:"expired_at"`
	Trial     bool      `json:"trial"`
	Banned    bool      `json:"banned"`
	Active    bool      `json:"active"`

	PaymentID *string `json:"payment_id"`

	Email *string `json:"email"`
}
