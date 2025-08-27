package models

type PaymentPlan struct {
	ID    int64   `gorm:"primaryKey" json:"id"`
	Price float64 `json:"price"`
	Desc  string  `json:"desc"`
	Link  string  `json:"link"`
	Days  int     `json:"days"`

	Active bool `json:"active"`
}
