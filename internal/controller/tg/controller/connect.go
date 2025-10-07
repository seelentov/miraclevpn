package controller

import (
	"fmt"
	"miraclevpn/internal/services/servers"
	"strconv"
	"strings"

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

	servers, err := c.srv.GetBest()
	if err != nil {
		panic(err)
	}

	text := "🌍 *Выберите сервер*\n\n"

	rows := make([][]tgbotapi.InlineKeyboardButton, 0)

	for _, server := range servers {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(server.RegionName, fmt.Sprintf("/connect:%v:%v", chatID, server.ID)),
			),
		)
	}

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Все сервера", fmt.Sprintf("/servers_all:%v", chatID)),
		),
	)

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", fmt.Sprintf("/stats:%v", chatID)),
		),
	)

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

func (c *ConnectTGController) GetConfig(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	chatID := data["chat_id"].(int64)
	userID := strconv.Itoa(int(chatID))
	serverIDStr := data["param"].(string)
	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		panic(err)
	}

	server, err := c.srv.GetServerByID(serverID)
	if err != nil {
		panic(err)
	}

	if server.Preview {
		panic("preview")
	}

	config, err := c.srv.GetConfig(userID, serverID)
	if err != nil {
		panic(err)
	}

	text := fmt.Sprintf("⏳ *Подключаемся к %s...*\n\n"+
		"📖 *Простая инструкция:*\n\n"+
		"1️⃣ *Скачайте приложение* OpenVPN Connect, если у вас его еще нет:\n"+
		"   - [Скачать для iOS](https://apps.apple.com/app/openvpn-connect/id590379981)\n"+
		"   - [Скачать для Android](https://play.google.com/store/apps/details?id=net.openvpn.openvpn)\n"+
		"   - [Скачать для ПК](https://openvpn.net/client)\n\n"+
		"2️⃣ *Используйте конфигурационный файл в приложении:*\n"+
		"   📎 `config.ovpn`\n\n"+
		"3️⃣ *Приложение откроется — подтвердите подключение.*\n\n"+
		"⚠️ *Важно!*\n"+
		"- Этот файл **одноразовый**.\n"+
		"- Если соединение прервалось, просто вернитесь сюда и скачайте новый.",
		server.RegionName)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("♻️ Обновить", fmt.Sprintf("/connect:%v:%v", chatID, server.ID)),
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

func (c *ConnectTGController) GetAll(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	chatID := data["chat_id"].(int64)

	servers, err := c.srv.GetAllServers()
	if err != nil {
		panic(err)
	}

	text := "🌍 *Выберите сервер*\n\n"

	rows := make([][]tgbotapi.InlineKeyboardButton, 0)

	for _, server := range servers {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(server.RegionName+" "+server.Host, fmt.Sprintf("/connect:%v:%v", chatID, server.ID)),
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

func (c *ConnectTGController) GetStats(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	chatID := data["chat_id"].(int64)

	servers, err := c.srv.GetAllServers()
	if err != nil {
		panic(err)
	}

	textB := strings.Builder{}

	textB.WriteString(" *📊 Статистика серверов*\n\n")

	for _, srv := range servers {
		server, currentUsersCount, err := c.srv.GetServerStatus(srv.ID)
		if err != nil {
			continue
		}

		textB.WriteString(server.RegionName)
		textB.WriteString(" ")
		textB.WriteString(server.Host)
		textB.WriteString(" - ")
		textB.WriteString(strconv.Itoa(currentUsersCount))
		textB.WriteString("/")
		textB.WriteString(strconv.Itoa(server.MaxUsers))
		textB.WriteString("\n")
	}

	msg := tgbotapi.NewMessage(chatID, textB.String())
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}
