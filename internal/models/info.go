package models

type Info struct {
	ID     int64  `gorm:"primaryKey" json:"id"`
	Slug   string `gorm:"uniqueIndex" json:"slug"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Active bool   `json:"active"`
}
