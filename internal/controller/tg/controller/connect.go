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

// Index handles /servers — shows the list of all servers with live status.
func (c *ConnectTGController) Index(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	chatID := data["chat_id"].(int64)
	u := data["user"].(*models.User)

	if c.blocked(bot, chatID, u) {
		return
	}

	srvList, err := c.srv.GetAllServersWithStatus()
	if err != nil {
		panic(err)
	}

	if len(srvList) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Нет доступных серверов"))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, s := range srvList {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				serverBtnLabel(s),
				fmt.Sprintf("/connect:%v:%v", chatID, s.Server.ID),
			),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, "🌍 *Выберите сервер для подключения:*")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}

// QuickConnect handles /quick_connect — picks the best available server automatically.
func (c *ConnectTGController) QuickConnect(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	chatID := data["chat_id"].(int64)
	userID := strconv.Itoa(int(chatID))
	u := data["user"].(*models.User)

	if c.blocked(bot, chatID, u) {
		return
	}

	best, err := c.srv.GetBestAvailableServer()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID,
			"⚠️ Нет доступных серверов. Попробуйте позже или выберите вручную — /servers"))
		return
	}

	config, err := c.srv.GetConfig(userID, best.Server.ID)
	if err != nil {
		panic(err)
	}

	c.sendConfig(bot, chatID, best.Server, config)
}

// Connect handles /connect:chatID:serverID — validates capacity and sends the VPN config file.
func (c *ConnectTGController) Connect(bot *tgbotapi.BotAPI, data map[string]interface{}) {
	chatID := data["chat_id"].(int64)
	userID := strconv.Itoa(int(chatID))
	u := data["user"].(*models.User)

	if c.blocked(bot, chatID, u) {
		return
	}

	serverID, err := strconv.ParseInt(data["param"].(string), 10, 64)
	if err != nil {
		panic(err)
	}

	server, online, err := c.srv.GetServerStatus(serverID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Сервер недоступен. Выберите другой — /servers"))
		return
	}

	if server.Preview {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Этот сервер недоступен"))
		return
	}

	if server.MaxUsers > 0 && online >= server.MaxUsers {
		bot.Send(tgbotapi.NewMessage(chatID,
			fmt.Sprintf("🔒 Сервер *%s* заполнен (%d/%d).\nВыберите другой — /servers",
				serverDisplayName(server), online, server.MaxUsers)))
		return
	}

	config, err := c.srv.GetConfig(userID, server.ID)
	if err != nil {
		panic(err)
	}

	c.sendConfig(bot, chatID, server, config)
}

func (c *ConnectTGController) sendConfig(bot *tgbotapi.BotAPI, chatID int64, server *models.Server, config string) {
	meta := vpnMeta(server.Type)

	desktopLine := ""
	if meta.windowsURL != "" {
		desktopLine = fmt.Sprintf("   - [Скачать для Windows](%s)\n"+
			"   - [Скачать для macOS](%s)\n", meta.windowsURL, meta.macosURL)
	}
	text := fmt.Sprintf("⏳ *Подключаемся к %s (%s)...*\n\n"+
		"📖 *Простая инструкция:*\n\n"+
		"1️⃣ *Скачайте приложение* %s, если у вас его еще нет:\n"+
		"   - [Скачать для iOS](%s)\n"+
		"   - [Скачать для Android](%s)\n"+
		"%s"+
		"\n2️⃣ *Откройте файл (%s) в приложении*\n\n"+
		"⚠️ Если не подключается получите новый файл, нажав *Обновить*.\n",
		server.RegionName, serverDisplayName(server),
		meta.appName, meta.iosURL, meta.androidURL, desktopLine, meta.fileName)

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("♻️ Обновить",
				fmt.Sprintf("/connect:%v:%v", chatID, server.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📱 iOS", meta.iosURL),
			tgbotapi.NewInlineKeyboardButtonURL("🤖 Android", meta.androidURL),
		),
	}
	if meta.windowsURL != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🖥 Windows", meta.windowsURL),
			tgbotapi.NewInlineKeyboardButtonURL(" macOS", meta.macosURL),
		))
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	docMsg := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
		Name:  meta.fileName,
		Bytes: []byte(config),
	})
	docMsg.Caption = text
	docMsg.ParseMode = "Markdown"
	docMsg.ReplyMarkup = keyboard

	if _, err := bot.Send(docMsg); err != nil {
		panic(err)
	}
}

func (c *ConnectTGController) blocked(bot *tgbotapi.BotAPI, chatID int64, u *models.User) bool {
	if u.ExpiredAt.Before(time.Now()) {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Подписка истекла"))
		return true
	}
	if u.Banned {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Аккаунт заблокирован"))
		return true
	}
	if !u.Active {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Аккаунт деактивирован"))
		return true
	}
	return false
}

func serverBtnLabel(s *servers.ServerWithStatus) string {
	typeLabel := serverTypeLabel(s.Server.Type)

	var prefix string
	switch {
	case !s.Available:
		prefix = "❌"
	case s.Server.MaxUsers > 0 && s.Online >= s.Server.MaxUsers:
		prefix = "🔒"
	default:
		prefix = "▶️"
	}

	var capacity string
	if s.Server.MaxUsers > 0 {
		capacity = fmt.Sprintf(" · 👥 %d/%d", s.Online, s.Server.MaxUsers)
	} else {
		capacity = fmt.Sprintf(" · 👥 %d", s.Online)
	}

	flag := s.Server.RegionFlagURL
	if flag != "" {
		flag += " "
	}

	return fmt.Sprintf("%s %s%s · %s%s",
		prefix, flag, serverDisplayName(s.Server), typeLabel, capacity)
}

func serverDisplayName(s *models.Server) string {
	if s.Name != "" {
		return s.Name
	}
	return s.RegionName
}

func serverTypeLabel(t string) string {
	switch t {
	case models.ServerTypeAmneziaWG:
		return "AWG"
	default:
		return "OVPN"
	}
}

type vpnMetadata struct {
	fileName   string
	appName    string
	iosURL     string
	androidURL string
	windowsURL string
	macosURL   string
}

func vpnMeta(serverType string) vpnMetadata {
	switch serverType {
	case models.ServerTypeAmneziaWG:
		return vpnMetadata{
			fileName:   "config.conf",
			appName:    "AmneziaVPN",
			iosURL:     "https://apps.apple.com/us/app/amneziawg/id6478942365",
			androidURL: "https://play.google.com/store/apps/details?id=org.amnezia.awg&hl=ru&pli=1",
			windowsURL: "https://github.com/amnezia-vpn/amnezia-client/releases/download/4.8.14.5/AmneziaVPN_4.8.14.5_x64.exe",
			macosURL:   "https://github.com/amnezia-vpn/amnezia-client/releases/download/4.8.14.5/AmneziaVPN_4.8.14.5_macos.pkg",
		}
	default:
		return vpnMetadata{
			fileName:   "config.ovpn",
			appName:    "OpenVPN Connect",
			iosURL:     "https://apps.apple.com/app/openvpn-connect/id590379981",
			androidURL: "https://play.google.com/store/apps/details?id=net.openvpn.openvpn",
		}
	}
}
