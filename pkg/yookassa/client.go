// Package yookassa provides YooKassa utilities for the application.
package yookassa

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"miraclevpn/internal/services/payment"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRequest = errors.New("request error")
)

var (
	URL = "https://api.yookassa.ru/v3/payments"
)

type Client struct {
	client    *http.Client
	shopID    string
	secret    string
	returnURL string
}

func NewClient(shopID, secret, returnURL string) *Client {
	return &Client{
		client:    &http.Client{},
		shopID:    shopID,
		secret:    secret,
		returnURL: returnURL,
	}
}

func (c *Client) CreatePayment(email string, description string, items []*payment.PaymentItem, paymentToken string, getReceipt bool, paymentMethodID string) (ID string, paymentURL string, err error) {
	sum := 0.0
	for _, it := range items {
		sum += it.Value * float64(it.Quantity)
	}

	payment := createPaymentRequest{
		Amount: amount{
			Value:    strconv.Itoa(int(sum)),
			Currency: string(items[0].Currency),
		},
		Capture:     true,
		Description: description,
		MetaData: map[string]string{
			"token": paymentToken,
		},
	}

	if paymentMethodID == "" {
		payment.Confirmation = confirmation{
			Type:      "redirect",
			ReturnURL: c.returnURL,
		}
	} else {
		payment.PaymentMethodID = paymentMethodID
	}

	if getReceipt {
		payment.Receipt = &createPaymentRequestReceipt{
			Customer: customer{
				Email: email,
			},
			Items: make([]item, 0, len(items)),
		}

		for _, it := range items {
			payment.Receipt.Items = append(payment.Receipt.Items, item{
				Description: it.Name,
				Quantity:    it.Quantity,
				Amount: amount{
					Value:    strconv.Itoa(int(it.Value * float64(it.Quantity))),
					Currency: string(it.Currency),
				},
				VatCode: int(it.Vat),
			})
		}
	}

	jsonData, err := json.Marshal(payment)

	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", err
	}

	req.SetBasicAuth(c.shopID, c.secret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", uuid.New().String())

	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("%w: %s: %s: %s", ErrRequest, resp.Status, string(body), string(jsonData))
	}

	var result createPaymentResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	return result.ID, result.Confirmation.ConfirmationURL, nil
}

type WebHookRes struct {
	Event  string  `json:"event"`
	Object Payment `json:"object"`
	Type   string  `json:"type"`
	/*
			"payment_method" : {
		Sep 23 07:24:05 5559569-nz15454 api[1808512]:       "type" : "yoo_money",
		Sep 23 07:24:05 5559569-nz15454 api[1808512]:       "id" : "3064349b-000f-5001-8000-15cf5edc0bb6",
		Sep 23 07:24:05 5559569-nz15454 api[1808512]:       "saved" : true,
		Sep 23 07:24:05 5559569-nz15454 api[1808512]:       "status" : "active",
		Sep 23 07:24:05 5559569-nz15454 api[1808512]:       "title" : "YooMoney wallet 410011758831136",
		Sep 23 07:24:05 5559569-nz15454 api[1808512]:       "account_number" : "410011758831136"
		Sep 23 07:24:05 5559569-nz15454 api[1808512]:     },
	*/
}

type Payment struct {
	Amount         Amount            `json:"amount"`
	CapturedAt     time.Time         `json:"captured_at"`
	CreatedAt      time.Time         `json:"created_at"`
	Description    string            `json:"description"`
	ID             string            `json:"id"`
	IncomeAmount   Amount            `json:"income_amount"`
	Metadata       map[string]string `json:"metadata"`
	Paid           bool              `json:"paid"`
	PaymentMethod  PaymentMethod     `json:"payment_method"`
	Recipient      Recipient         `json:"recipient"`
	Refundable     bool              `json:"refundable"`
	RefundedAmount Amount            `json:"refunded_amount"`
	Status         string            `json:"status"`
	Test           bool              `json:"test"`
}

type Amount struct {
	Currency string  `json:"currency"`
	Value    float64 `json:"string"`
}

type PaymentMethod struct {
	AccountNumber string `json:"account_number"`
	ID            string `json:"id"`
	Saved         bool   `json:"saved"`
	Status        string `json:"status"`
	Title         string `json:"title"`
	Type          string `json:"type"`
}

type Recipient struct {
	AccountID string `json:"account_id"`
	GatewayID string `json:"gateway_id"`
}

type createPaymentRequest struct {
	Amount          amount                       `json:"amount"`
	Capture         bool                         `json:"capture"`
	Description     string                       `json:"description"`
	Confirmation    confirmation                 `json:"confirmation,omitzero"`
	Receipt         *createPaymentRequestReceipt `json:"receipt,omitempty"`
	PaymentMethodID string                       `json:"payment_method_id,omitempty"`
	MetaData        map[string]string            `json:"metadata"`
}

type amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type confirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

type createPaymentRequestReceipt struct {
	Customer customer `json:"customer"`
	Items    []item   `json:"items"`
}

type customer struct {
	Email string `json:"email"`
}

type item struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Amount      amount `json:"amount"`
	VatCode     int    `json:"vat_code"`
}

type createPaymentResponse struct {
	ID           string               `json:"id"`
	Status       string               `json:"status"`
	Amount       amount               `json:"amount"`
	Description  string               `json:"description"`
	Recipient    recipient            `json:"recipient"`
	CreatedAt    string               `json:"created_at"`
	Confirmation responseConfirmation `json:"confirmation"`
	Test         bool                 `json:"test"`
	Paid         bool                 `json:"paid"`
	Refundable   bool                 `json:"refundable"`
	Metadata     map[string]string    `json:"metadata"`
}

type recipient struct {
	AccountID string `json:"account_id"`
	GatewayID string `json:"gateway_id"`
}

type responseConfirmation struct {
	Type            string `json:"type"`
	ConfirmationURL string `json:"confirmation_url"`
}
