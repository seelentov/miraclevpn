package models

import "time"

type User struct {
	ID        int64 `gorm:"primaryKey"`
	ExpiredAt time.Time
	Trial     bool
	Banned    bool
}
