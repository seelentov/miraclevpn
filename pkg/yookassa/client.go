// Package yookassa provides YooKassa utilities for the application.
package yookassa

import "net/http"

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{},
	}
}
