package tg

import (
	"log"
	"strconv"
	"strings"

	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TgDaemon struct {
	botToken string
	jwtSrv   *crypt.JwtService
	userRepo *repo.UserRepository
}

func NewTgDaemon(botToken string, jwtSrv *crypt.JwtService, userRepo *repo.UserRepository) *TgDaemon {
	return &TgDaemon{
		botToken: botToken,
		jwtSrv:   jwtSrv,
		userRepo: userRepo,
	}
}

func (d *TgDaemon) Start() {
	bot, err := tgbotapi.NewBotAPI(d.botToken)
	if err != nil {
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
			}

			chatID := update.Message.Chat.ID
			text := update.Message.Text

			if strings.HasPrefix(text, "/start ") {
				token := strings.TrimPrefix(text, "/start ")
				claims, err := d.jwtSrv.ParseToken(token)
				if err != nil || claims.UserID == "" {
					msg := tgbotapi.NewMessage(chatID, "Некорректная ссылка или токен.")
					bot.Send(msg)
					continue
				}

				userID, _ := strconv.ParseInt(claims.UserID, 10, 64)
				err = d.userRepo.SetTGChatID(userID, chatID)
				if err != nil {
					msg := tgbotapi.NewMessage(chatID, "Ошибка активации. Попробуйте позже.")
					bot.Send(msg)
					continue
				}

				msg := tgbotapi.NewMessage(chatID, "Вы успешно активировали Telegram!")
				bot.Send(msg)
				continue
			}

			msg := tgbotapi.NewMessage(chatID, "Я бот, используйте ссылку для активации.")
			bot.Send(msg)
		}
	}()
}
