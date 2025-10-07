// Package tg provides Telegram bot utilities for the application.
package tg

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	ErrRequest = errors.New("request error")
)

type Client struct {
	token  string
	name   string
	client *http.Client
}

func NewClient(token string, name string) *Client {
	return &Client{
		token,
		name,
		&http.Client{},
	}
}

func (c *Client) SendMessage(to string, message string) error {
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s", c.token, to, strings.ReplaceAll(message, " ", "+")))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: %s: %s", ErrRequest, resp.Status, string(body))
	}

	return nil
}

func (c *Client) GetName() string {
	return c.name
}

func (c *Client) GetStatus() (bool, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getMe", c.token))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("%w: %s: %s", ErrRequest, resp.Status, string(body))
	}

	return true, nil
}
