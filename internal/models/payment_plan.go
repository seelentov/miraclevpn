package models

type PaymentPlan struct {
	ID      int64   `gorm:"primaryKey" json:"id"`
	Price   float64 `json:"price"`
	Desc    string  `json:"desc"`
	PayDesc string  `json:"pay_desc"`
	Link    string  `json:"link"`
	Days    int     `json:"days"`

	Currency Currency `json:"currency"`
	VatCode  VatCode  `json:"vat_code"`

	Active bool `json:"active"`
}
