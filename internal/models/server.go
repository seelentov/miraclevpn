package models

type Server struct {
	ID int64 `gorm:"primaryKey"`

	Host   string `gorm:"uniqueIndex"`
	Region string

	Active bool
}
