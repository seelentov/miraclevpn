// Package controller provides control functions for Telegram bot handlers.
package controller

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AuthTGController struct{}

func NewAuthTGController() *AuthTGController {
	return &AuthTGController{}
}

func (c *AuthTGController) GetToken(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	token := data["token"].(string)
	chatID := data["chat_id"].(int64)

	text := fmt.Sprintf("🔐 *Ваш ключ:* `%s`\n\n", token)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}
