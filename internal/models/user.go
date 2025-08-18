package models

import "time"

type User struct {
	ID int64 `gorm:"primaryKey"`

	Phone    string
	Password string

	TGChat int64

	Expiration time.Time
	Active     bool
}
