package models

import "time"

type User struct {
	ID int64 `gorm:"primaryKey"`

	Username string `gorm:"uniqueIndex"`
	Password string

	TGChat *int64 `gorm:"uniqueIndex"`

	ExpiredAt time.Time
	Active    bool
}
