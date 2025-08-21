package tg

import (
	"log"
	"strconv"

	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type TgDaemon struct {
	botToken string
	jwtSrv   *crypt.JwtService
	userRepo *repo.UserRepository

	logger *zap.Logger

	stopChan chan struct{}
}

func NewTgDaemon(botToken string, jwtSrv *crypt.JwtService, userRepo *repo.UserRepository, logger *zap.Logger) *TgDaemon {
	return &TgDaemon{
		botToken: botToken,
		jwtSrv:   jwtSrv,
		userRepo: userRepo,
		logger:   logger,
		stopChan: make(chan struct{}),
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

			token := text
			_ = tgbotapi.NewMessage(chatID, text)
			claims, err := d.jwtSrv.ParseToken(token)
			if err != nil || claims.UserID == "" {
				d.logger.Error("failed to parse JWT token", zap.String("token", token), zap.Error(err))
				msg := tgbotapi.NewMessage(chatID, "Некорректная ссылка или токен.")
				bot.Send(msg)
				continue
			}

			userID, _ := strconv.ParseInt(claims.UserID, 10, 64)
			err = d.userRepo.SetTGChatID(userID, chatID)
			if err != nil {
				d.logger.Error("failed to set TG chat ID", zap.Int64("user_id", userID), zap.Error(err))
				msg := tgbotapi.NewMessage(chatID, "Ошибка активации. Попробуйте позже.")

				bot.Send(msg)
				continue
			}

			err = d.userRepo.Activate(userID)
			if err != nil {
				d.logger.Error("failed to activate user", zap.Int64("user_id", userID), zap.Error(err))
				msg := tgbotapi.NewMessage(chatID, "Ошибка активации. Попробуйте позже.")
				bot.Send(msg)
				continue
			}

			msg := tgbotapi.NewMessage(chatID, "Вы успешно активировали аккаунт!")
			bot.Send(msg)
		}
	}()
}

func (d *TgDaemon) Stop() {
	close(d.stopChan)
}
