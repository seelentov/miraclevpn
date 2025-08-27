package models

import "time"

type UserServer struct {
	UserID   string `gorm:"primaryKey" json:"user_id"`
	ServerID int64  `gorm:"primaryKey" json:"server_id"`

	Config     string `json:"config"`
	ConfigFile string `json:"config_file"`

	UpdatedAt time.Time `json:"updated_at"`
}
