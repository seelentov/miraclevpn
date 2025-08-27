package models

type Requests struct {
	ID     int64 `gorm:"primaryKey"`
	UserID string

	Region string
}
