package middleware

import (
	"fmt"
	"miraclevpn/internal/controller/tg"
	"miraclevpn/internal/services/sender"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

func RecoverrHandler(debug bool, sender sender.Sender, adminTo string, logger *zap.Logger) tg.Recoverr {
	return func(bot *tgbotapi.BotAPI, data map[string]interface{}, err interface{}) {
		logger.Error("Panic recovered", zap.Any("error", err), zap.String("path", data["path"].(string)), zap.Int64("user_id", data["chat_id"].(int64)))

		if !debug {
			sender.SendMessage(adminTo, fmt.Sprintf("%v", err))
		}

		bot.Send(tgbotapi.NewMessage(data["chat_id"].(int64), "❌ Ошибка при выполнении команды"))
	}

}
