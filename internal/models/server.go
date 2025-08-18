package models

type Server struct {
	ID int64 `gorm:"primaryKey"`

	Host   string
	Region string

	Active bool
}
