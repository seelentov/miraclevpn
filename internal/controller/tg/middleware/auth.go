// Package middleware provides middleware functions for Telegram bot handlers.
package middleware

import (
	"miraclevpn/internal/controller/tg"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/auth"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func AuthMiddlewareTg(authSrv *auth.AuthService, userRepo *repo.UserRepository) tg.Handler {
	return func(bot *tgbotapi.BotAPI, data map[string]interface{}) {
		uid := strconv.Itoa(int(data["chat_id"].(int64)))

		token, err := authSrv.Authenticate(uid, map[string]interface{}{
			"telegram": true,
		}, false)

		if err != nil {
			panic(err)
		}

		user, err := userRepo.FindByID(uid)
		if err != nil {
			panic(err)
		}

		data["user"] = user
		data["user_id"] = user.ID
		data["token"] = token
	}
}
