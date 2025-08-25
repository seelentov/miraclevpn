package models

import "time"

type UserServer struct {
	UserID   string `gorm:"primaryKey"`
	ServerID int64  `gorm:"primaryKey"`

	Config     string
	ConfigFile string

	UpdatedAt time.Time
}
