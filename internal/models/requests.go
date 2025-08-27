package models

type Requests struct {
	ID     int64  `gorm:"primaryKey" json:"id"`
	UserID string `json:"user_id"`

	Region string `json:"region"`
}
