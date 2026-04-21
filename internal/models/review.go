package models

type Review struct {
	ID       int64  `gorm:"primaryKey" json:"id"`
	Name     string `json:"name"`
	Text     string `json:"text"`
	PhotoURL string `json:"photo_url"`
	URL      string `json:"url"`
	Active   bool   `json:"active"`
	SortOrder int   `gorm:"default:0" json:"sort_order"`
}
