package models

import "time"

type User struct {
	ID int64 `gorm:"primaryKey"`

	Phone    string `gorm:"uniqueIndex"`
	Password string

	TGChat *int64 `gorm:"uniqueIndex"`

	ExpiredAt time.Time
	Active    bool
}
