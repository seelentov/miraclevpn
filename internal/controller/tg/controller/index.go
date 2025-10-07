// Package controller provides control functions for Telegram bot handlers.
package controller

import (
	"fmt"
	"miraclevpn/internal/models"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type IndexTGController struct {
	paymentPageURL string
}

func NewIndexTGController(paymentPageURL string) *IndexTGController {
	return &IndexTGController{paymentPageURL}
}

func (c *IndexTGController) Index(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	user := data["user"].(*models.User)
	token := data["token"].(string)
	chatID := data["chat_id"].(int64)

	daysLeft := int(time.Until(user.ExpiredAt).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}

	text := fmt.Sprintf("🔐 *Ваш VPN-профиль*\n\n"+
		"🆔 *UID:* `%s`\n"+
		"✅ *Статус подписки:* Активна до *%s* (осталось *%d* дней)\n\n"+
		"*Управление:*",
		user.ID,
		user.ExpiredAt.Format("02.01.2006"),
		daysLeft)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("♻️ Продлить подписку", c.paymentPageURL+"/?token="+token),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Подключиться", fmt.Sprintf("/servers:%v", chatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 Получить ключ", fmt.Sprintf("/get_key:%v", chatID)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}
