package controller

import (
	"fmt"
	"miraclevpn/internal/models"
	"miraclevpn/internal/services/servers"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ConnectTGController struct {
	srv *servers.ServersService
}

func NewConnectTGController(srv *servers.ServersService) *ConnectTGController {
	return &ConnectTGController{srv}
}

func (c *ConnectTGController) Index(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	chatID := data["chat_id"].(int64)
	userID := strconv.Itoa(int(chatID))

	u := data["user"].(*models.User)

	if u.ExpiredAt.Before(time.Now()) {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Подписка истекла"))
		return
	}

	if u.Banned {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Аккаунт заблокирован"))

		return
	}

	if !u.Active {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Аккаунт деактивирован"))
		return
	}

	bestServer, err := c.srv.GetOnlyBest()
	if err != nil {
		panic(err)
	}

	server, err := c.srv.GetServerByID(bestServer.ID)
	if err != nil {
		panic(err)
	}

	if server.Preview {
		panic("preview")
	}

	config, err := c.srv.GetConfig(userID, server.ID)
	if err != nil {
		panic(err)
	}

	text := fmt.Sprintf("⏳ *Подключаемся к %s...*\n\n"+
		"📖 *Простая инструкция:*\n\n"+
		"1️⃣ *Скачайте приложение* OpenVPN Connect, если у вас его еще нет:\n"+
		"   - [Скачать для iOS](https://apps.apple.com/app/openvpn-connect/id590379981)\n"+
		"   - [Скачать для Android](https://play.google.com/store/apps/details?id=net.openvpn.openvpn)\n\n"+
		"2️⃣ *Откройте файл (config.ovpn) в приложении*\n\n"+
		"⚠️ Если не подключается получите новый файл, нажав **Обновить**.\n",
		server.RegionName)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("♻️ Обновить", fmt.Sprintf("/connect:%v:%v", chatID, server.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("ℹ️ Инструкция IOS", "https://miiboost.ru/ios.mp4"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("ℹ️ Инструкция Android", "https://miiboost.ru/android.mp4"),
		),
	)

	fileBytes := tgbotapi.FileBytes{
		Name:  "config.ovpn",
		Bytes: []byte(config),
	}

	docMsg := tgbotapi.NewDocument(chatID, fileBytes)
	docMsg.Caption = text
	docMsg.ParseMode = "Markdown"
	docMsg.ReplyMarkup = keyboard

	if _, err := bot.Send(docMsg); err != nil {
		panic(err)
	}
}
