package middleware

import (
	"miraclevpn/internal/controller/tg"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func NotFoundHandler() tg.Handler {
	return func(bot *tgbotapi.BotAPI, data map[string]interface{}) {
		bot.Send(tgbotapi.NewMessage(data["chat_id"].(int64), "❌ Неверная команда"))
	}

}
