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
			tgbotapi.NewInlineKeyboardButtonData("🌍 Подключиться", fmt.Sprintf("/servers:%v", chatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 7 дней подписки за отзыв!", fmt.Sprintf("/gift:%v", chatID)),
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

func (c *IndexTGController) FreeForReview(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	user := data["user"].(*models.User)
	chatID := data["chat_id"].(int64)

	text := "🎁 *7 дней подписки за отзыв\\!*\n\n" +
		"Получите *7 дней бесплатной подписки* в обмен на ваш честный отзыв\\!\n\n" +
		"*Как это работает:*\n" +
		"1\\. Напишите отзыв о нашем сервисе в диалоге с @miiboost\\_support\n" +
		"2\\. Обязательно укажите в отзыве ваш UID: `" + user.ID + "`\n" +
		"3\\. После проверки отзыва мы активируем для вас 7 дней бесплатной подписки\\!\n\n" +
		"Ваш отзыв поможет нам стать лучше\\! ❤️\n\n" +
		"\\* Воспользоваться акцией возможно только 1 раз на аккаунт"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("💬 Написать отзыв", "https://t.me/miiboost_support"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", fmt.Sprintf("/start:%v", chatID)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "MarkdownV2"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}
