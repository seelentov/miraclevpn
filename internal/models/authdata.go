package models

import "time"

type AuthData struct {
	ID   int64     `gorm:"primaryKey" json:"id"`
	UID  string    `json:"uid"`
	Data JSONB     `json:"data" gorm:"type:jsonb"`
	Date time.Time `json:"date"`
}
