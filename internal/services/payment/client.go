// Package payment provides payment logic
package payment

import (
	"miraclevpn/internal/models"
)

type PaymentItem struct {
	Name     string
	Quantity int
	Value    float64
	Currency models.Currency
	Vat      models.VatCode
}

type PaymentClient interface {
	CreatePayment(email string, description string, items []*PaymentItem, getReceipt bool, paymentMethodID string, meta map[string]string) (ID string, paymentURL string, err error)
}
