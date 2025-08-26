package models

type News struct {
	ID int64 `gorm:"primaryKey"`

	Title string
	Text  string

	Active bool
}
