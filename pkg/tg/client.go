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
	client *http.Client
}

func NewTgClient(token string) *TgClient {
	return &TgClient{
		token,
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
