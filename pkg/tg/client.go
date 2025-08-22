// Package tg provides Telegram bot utilities for the application.
package tg

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	ErrRequest = errors.New("request error")
)

type TgClient struct {
	token  string
	name   string
	client *http.Client
}

func NewTgClient(token string, name string) *TgClient {
	return &TgClient{
		token,
		name,
		&http.Client{},
	}
}

func (c *TgClient) SendMessage(to string, message string) error {
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s", c.token, to, message))
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

func (c *TgClient) GetName() string {
	return c.name
}

func (c *TgClient) GetStatus() (bool, error) {
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
