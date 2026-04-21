package controller

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/servers"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ChatButton struct {
	Text   string `json:"text"`
	Action string `json:"action,omitempty"`
	URL    string `json:"url,omitempty"`
}

type ChatFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ChatMsg struct {
	Text    string         `json:"text"`
	Buttons [][]ChatButton `json:"buttons,omitempty"`
	File    *ChatFile      `json:"file,omitempty"`
}

type ChatController struct {
	userRepo    *repo.UserRepository
	serversSrv  *servers.ServersService
	jwtSrv      *crypt.JwtService
	jwtDuration time.Duration
	paymentURL  string
	lkURL       string
	logger      *zap.Logger
}

func NewChatController(
	userRepo *repo.UserRepository,
	serversSrv *servers.ServersService,
	jwtSrv *crypt.JwtService,
	jwtDuration time.Duration,
	paymentURL string,
	lkURL string,
	logger *zap.Logger,
) *ChatController {
	return &ChatController{userRepo, serversSrv, jwtSrv, jwtDuration, paymentURL, lkURL, logger}
}

func (c *ChatController) GetPage(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "chat.html", nil)
}

// POST /api/chat/start
// Body: {"type":"new"} or {"type":"key","key":"JWT"}
func (c *ChatController) Start(ctx *gin.Context) {
	var req struct {
		Type string `json:"type"`
		Key  string `json:"key"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var userID, token string

	switch req.Type {
	case "new":
		uid, err := generateChatUID()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate uid"})
			return
		}
		if _, err := c.userRepo.Create(uid); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
		t, err := c.jwtSrv.GenerateToken(map[string]string{"user_id": uid}, c.jwtDuration)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}
		userID = uid
		token = t
	case "key":
		claims, err := c.jwtSrv.ParseToken(req.Key)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный ключ"})
			return
		}
		userID = claims.Data["user_id"]
		if _, err := c.userRepo.FindByID(userID); err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не найден"})
			return
		}
		token = req.Key
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'new' or 'key'"})
		return
	}

	user, err := c.userRepo.FindByID(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"token":    token,
		"messages": []ChatMsg{c.buildMainMenu(user, token)},
	})
}

// POST /api/chat/action
// Header: Authorization: Bearer <token>
// Body: {"action":"menu"|"servers"|"quick_connect"|"connect"|"get_key"|"gift", "server_id":0}
func (c *ChatController) Action(ctx *gin.Context) {
	token := chatExtractToken(ctx)
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	claims, err := c.jwtSrv.ParseToken(token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	userID := claims.Data["user_id"]

	user, err := c.userRepo.FindByID(userID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var req struct {
		Action   string `json:"action"`
		ServerID int64  `json:"server_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// parse "connect:42" shorthand
	action := req.Action
	serverID := req.ServerID
	if strings.HasPrefix(action, "connect:") {
		parts := strings.SplitN(action, ":", 2)
		serverID, _ = strconv.ParseInt(parts[1], 10, 64)
		action = "connect"
	}

	var msgs []ChatMsg
	switch action {
	case "menu":
		msgs = []ChatMsg{c.buildMainMenu(user, token)}
	case "get_key":
		msgs = []ChatMsg{{
			Text: fmt.Sprintf("🔐 *Ваш ключ:*\n\n`%s`\n\nСохраните его — он понадобится для входа с другого устройства.", token),
			Buttons: [][]ChatButton{
				{{Text: "🏠 Главное меню", Action: "menu"}},
			},
		}}
	case "servers":
		msgs = c.buildServersList(user)
	case "quick_connect":
		msgs = c.buildQuickConnect(user, userID, token)
	case "connect":
		msgs = c.buildConnect(user, userID, serverID, token)
	default:
		msgs = []ChatMsg{{Text: "⚠️ Неизвестное действие"}}
	}

	ctx.JSON(http.StatusOK, gin.H{"messages": msgs})
}

// GET /api/chat/dl/:server_id?token=...
func (c *ChatController) GetConfig(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		token = chatExtractToken(ctx)
	}
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	claims, err := c.jwtSrv.ParseToken(token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	userID := claims.Data["user_id"]

	serverIDStr := ctx.Param("server_id")
	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid server_id"})
		return
	}

	srv, err := c.serversSrv.GetServerByID(serverID)
	if err != nil || srv.Preview {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	config, err := c.serversSrv.GetConfig(userID, serverID)
	if err != nil {
		c.logger.Error("failed to get config", zap.String("user_id", userID), zap.Int64("server_id", serverID), zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	meta := chatVpnMeta(srv.Type)
	fileName := ctx.Query("name")
	if fileName == "" {
		fileName = fmt.Sprintf("vpn_%d%s", rand.Intn(1001), meta.fileExt)
	}
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.String(http.StatusOK, config)
}

func (c *ChatController) buildMainMenu(user *models.User, token string) ChatMsg {
	daysLeft := int(time.Until(user.ExpiredAt).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}
	status := "✅ Активна"
	if user.ExpiredAt.Before(time.Now()) {
		status = "❌ Истекла"
	}
	text := fmt.Sprintf("🔐 *Ваш VPN-профиль*\n\n🆔 *UID:* `%s`\n%s до *%s* (осталось *%d* дней)\n\n*Управление:*",
		user.ID, status, user.ExpiredAt.Format("02.01.2006"), daysLeft)

	buttons := [][]ChatButton{
		{{Text: "⚡ Быстрое подключение", Action: "quick_connect"}},
		{{Text: "🌍 Выбрать сервер", Action: "servers"}},
	}
	if c.paymentURL != "" {
		buttons = append(buttons, []ChatButton{{Text: "♻️ Продлить подписку", URL: c.paymentURL + "/?token=" + token}})
	}
	buttons = append(buttons, []ChatButton{{Text: "🔑 Получить ключ", Action: "get_key"}})
	if c.lkURL != "" && user.PaymentID != nil {
		buttons = append(buttons, []ChatButton{{Text: "❌ Отменить подписку", URL: c.lkURL + "/?token=" + token}})
	}

	return ChatMsg{Text: text, Buttons: buttons}
}

func (c *ChatController) buildServersList(user *models.User) []ChatMsg {
	if user.ExpiredAt.Before(time.Now()) || user.Banned || !user.Active {
		return c.blockedMsgs(user)
	}
	srvList, err := c.serversSrv.GetAllServersWithStatus()
	if err != nil || len(srvList) == 0 {
		return []ChatMsg{{
			Text:    "⚠️ Нет доступных серверов",
			Buttons: [][]ChatButton{{{Text: "🔙 Назад", Action: "menu"}}},
		}}
	}
	var rows [][]ChatButton
	for _, s := range srvList {
		if s.Server.Preview {
			continue
		}
		rows = append(rows, []ChatButton{{
			Text:   chatServerBtnLabel(s),
			Action: fmt.Sprintf("connect:%d", s.Server.ID),
		}})
	}
	if len(rows) == 0 {
		return []ChatMsg{{Text: "⚠️ Нет доступных серверов", Buttons: [][]ChatButton{{{Text: "🔙 Назад", Action: "menu"}}}}}
	}
	rows = append(rows, []ChatButton{{Text: "🔙 Назад", Action: "menu"}})
	return []ChatMsg{{Text: "🌍 *Выберите сервер для подключения:*", Buttons: rows}}
}

func (c *ChatController) buildQuickConnect(user *models.User, userID, token string) []ChatMsg {
	if user.ExpiredAt.Before(time.Now()) || user.Banned || !user.Active {
		return c.blockedMsgs(user)
	}
	best, err := c.serversSrv.GetBestAvailableServer()
	if err != nil {
		return []ChatMsg{{
			Text:    "⚠️ Нет доступных серверов. Попробуйте позже или выберите вручную.",
			Buttons: [][]ChatButton{{{Text: "🌍 Выбрать сервер", Action: "servers"}}, {{Text: "🔙 Назад", Action: "menu"}}},
		}}
	}
	return c.buildConfigMsg(best.Server, token)
}

func (c *ChatController) buildConnect(user *models.User, userID string, serverID int64, token string) []ChatMsg {
	if user.ExpiredAt.Before(time.Now()) || user.Banned || !user.Active {
		return c.blockedMsgs(user)
	}
	srv, online, err := c.serversSrv.GetServerStatus(serverID)
	if err != nil || srv.Preview {
		return []ChatMsg{{
			Text:    "⚠️ Сервер недоступен. Выберите другой.",
			Buttons: [][]ChatButton{{{Text: "🌍 Выбрать другой", Action: "servers"}}},
		}}
	}
	if srv.MaxUsers > 0 && online >= srv.MaxUsers {
		return []ChatMsg{{
			Text:    fmt.Sprintf("🔒 Сервер *%s* заполнен (%d/%d).\nВыберите другой.", chatServerDisplayName(srv), online, srv.MaxUsers),
			Buttons: [][]ChatButton{{{Text: "🌍 Выбрать другой", Action: "servers"}}},
		}}
	}
	return c.buildConfigMsg(srv, token)
}

func (c *ChatController) buildConfigMsg(srv *models.Server, token string) []ChatMsg {
	meta := chatVpnMeta(srv.Type)
	fileName := fmt.Sprintf("vpn_%d%s", rand.Intn(1001), meta.fileExt)
	fileURL := fmt.Sprintf("/api/chat/dl/%d?token=%s&name=%s", srv.ID, token, fileName)

	desktopLine := ""
	if meta.windowsURL != "" {
		desktopLine = fmt.Sprintf("\n   [Windows](%s) · [macOS](%s)", meta.windowsURL, meta.macosURL)
	}
	text := fmt.Sprintf(
		"⏳ *Подключаемся к %s (%s)...*\n\n📖 *Инструкция:*\n\n1️⃣ Скачайте приложение *%s*:\n   [iOS](%s) · [Android](%s)%s\n\n2️⃣ Нажмите *Скачать конфиг* ниже и откройте файл `%s` в приложении\n\n⚠️ Если не подключается — нажмите *Обновить*.",
		srv.RegionName, chatServerDisplayName(srv),
		meta.appName, meta.iosURL, meta.androidURL, desktopLine,
		fileName,
	)

	buttons := [][]ChatButton{
		{{Text: "♻️ Обновить", Action: fmt.Sprintf("connect:%d", srv.ID)}},
		{{Text: "📱 iOS", URL: meta.iosURL}, {Text: "🤖 Android", URL: meta.androidURL}},
	}
	if meta.windowsURL != "" {
		buttons = append(buttons, []ChatButton{
			{Text: "🖥 Windows", URL: meta.windowsURL},
			{Text: " macOS", URL: meta.macosURL},
		})
	}
	buttons = append(buttons, []ChatButton{{Text: "🔙 Меню", Action: "menu"}})

	return []ChatMsg{{
		Text:    text,
		Buttons: buttons,
		File: &ChatFile{
			Name: fileName,
			URL:  fileURL,
		},
	}}
}

func (c *ChatController) blockedMsgs(user *models.User) []ChatMsg {
	var text string
	switch {
	case user.Banned:
		text = "⚠️ Аккаунт заблокирован"
	case !user.Active:
		text = "⚠️ Аккаунт деактивирован"
	default:
		text = "⚠️ Подписка истекла. Продлите её, чтобы продолжить."
	}
	return []ChatMsg{{
		Text:    text,
		Buttons: [][]ChatButton{{{Text: "🏠 Меню", Action: "menu"}}},
	}}
}

func chatExtractToken(ctx *gin.Context) string {
	auth := ctx.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ctx.GetHeader("X-Chat-Token")
}

func generateChatUID() (string, error) {
	b := make([]byte, 8)
	if _, err := crand.Read(b); err != nil {
		return "", err
	}
	return "web-" + hex.EncodeToString(b), nil
}

type chatVpnMetadata struct {
	fileExt    string
	appName    string
	iosURL     string
	androidURL string
	windowsURL string
	macosURL   string
}

func chatVpnMeta(serverType string) chatVpnMetadata {
	switch serverType {
	case models.ServerTypeAmneziaWG:
		return chatVpnMetadata{
			fileExt:    ".conf",
			appName:    "AmneziaVPN",
			iosURL:     "https://apps.apple.com/us/app/amneziawg/id6478942365",
			androidURL: "https://play.google.com/store/apps/details?id=org.amnezia.awg&hl=ru&pli=1",
			windowsURL: "https://github.com/amnezia-vpn/amnezia-client/releases/download/4.8.14.5/AmneziaVPN_4.8.14.5_x64.exe",
			macosURL:   "https://github.com/amnezia-vpn/amnezia-client/releases/download/4.8.14.5/AmneziaVPN_4.8.14.5_macos.pkg",
		}
	default:
		return chatVpnMetadata{
			fileExt:    ".ovpn",
			appName:    "OpenVPN Connect",
			iosURL:     "https://apps.apple.com/app/openvpn-connect/id590379981",
			androidURL: "https://play.google.com/store/apps/details?id=net.openvpn.openvpn",
		}
	}
}

func chatServerBtnLabel(s *servers.ServerWithStatus) string {
	var prefix string
	switch {
	case !s.Available:
		prefix = "❌"
	case s.Server.MaxUsers > 0 && s.Online >= s.Server.MaxUsers:
		prefix = "🔒"
	default:
		prefix = "▶️"
	}
	typeLabel := chatServerTypeLabel(s.Server.Type)
	var capacity string
	if s.Server.MaxUsers > 0 {
		capacity = fmt.Sprintf(" · 👥 %d/%d", s.Online, s.Server.MaxUsers)
	} else {
		capacity = fmt.Sprintf(" · 👥 %d", s.Online)
	}
	return fmt.Sprintf("%s %s · %s%s", prefix, chatServerDisplayName(s.Server), typeLabel, capacity)
}

func chatServerDisplayName(s *models.Server) string {
	if s.Name != "" {
		return s.Name
	}
	return s.RegionName
}

func chatServerTypeLabel(t string) string {
	if t == models.ServerTypeAmneziaWG {
		return "AWG"
	}
	return "OVPN"
}
