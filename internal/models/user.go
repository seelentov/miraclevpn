package models

import "time"

type User struct {
	ID        string `gorm:"primaryKey"`
	ExpiredAt time.Time
	Trial     bool
	Banned    bool
}
