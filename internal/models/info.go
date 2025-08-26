package models

type Info struct {
	ID    int64  `gorm:"primaryKey"`
	Slug  string `gorm:"uniqueIndex"`
	Title string
	Text  string
}
