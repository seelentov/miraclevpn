package models

import "time"

type TgTemp struct {
	ID int64 `gorm:"primaryKey"`

	UserID  int64
	Message string

	Expired time.Time
}
