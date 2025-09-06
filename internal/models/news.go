package models

type News struct {
	ID int64 `gorm:"primaryKey" json:"id"`

	Title string `json:"title"`
	Text  string `json:"text"`

	Active bool `json:"active"`

	Repeat bool `json:"repeat"`
}
