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
	lkURL          string
}

func NewIndexTGController(paymentPageURL string, lkURL string) *IndexTGController {
	return &IndexTGController{paymentPageURL, lkURL}
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

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚡ Быстрое подключение", fmt.Sprintf("/quick_connect:%v", chatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Выбрать сервер", fmt.Sprintf("/servers:%v", chatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("♻️ Продлить подписку", c.paymentPageURL+"/?token="+token),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 Получить ключ", fmt.Sprintf("/get_key:%v", chatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("❓ Поддержка", "https://t.me/miiboost_support"),
		),
	}

	if user.PaymentID != nil {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("❌ Отменить подписку", c.lkURL+"/?token="+token),
			),
		)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		rows...,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}

