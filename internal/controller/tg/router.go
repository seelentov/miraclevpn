// Package tg implements a simple router for Telegram bot handlers with middleware support and error recovery.
package tg

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	ErrNotFound = tgbotapi.Error{Message: "not found"}
)

type Handler func(bot *tgbotapi.BotAPI, data map[string]interface{})
type Recoverr func(bot *tgbotapi.BotAPI, data map[string]interface{}, err interface{})

type Router struct {
	middlewares []Handler
	handlers    map[string]Handler

	notFound Handler
	recoverr Recoverr

	bot *tgbotapi.BotAPI
}

func NewRouter(token string) (*Router, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Router{
		middlewares: make([]Handler, 0),
		handlers:    make(map[string]Handler),
		notFound:    nil,
		recoverr: func(bot *tgbotapi.BotAPI, data map[string]interface{}, err interface{}) {
			log.Println(err)
			bot.Send(tgbotapi.NewMessage(data["chat_id"].(int64), fmt.Sprintf("%v", err)))
		},
		bot: bot,
	}, nil
}

func (r *Router) Use(middleware Handler) {
	r.middlewares = append(r.middlewares, middleware)
}

func (r *Router) Use404(notFound Handler) {
	r.notFound = notFound
}

func (r *Router) UseRecover(recoverr Recoverr) {
	r.recoverr = recoverr
}

func (r *Router) UseHandler(path string, handler Handler) {
	r.handlers[path] = handler
}

func (r *Router) Handle(path string, data map[string]interface{}) {
	defer func() {
		if rec := recover(); rec != nil {
			r.recoverr(r.bot, data, rec)
		}
	}()

	if data == nil {
		data = make(map[string]interface{})
	}

	for _, middleware := range r.middlewares {
		middleware(r.bot, data)
	}

	handler, ok := r.handlers[path]
	if !ok {
		if r.notFound != nil {
			r.notFound(r.bot, data)
		}
		return
	}

	handler(r.bot, data)
}

func (r *Router) printHandlersInfo() {
	fmt.Println("=== Telegram Bot Router Information ===")
	fmt.Printf("Bot username: @%s\n", r.bot.Self.UserName)

	fmt.Printf("\n📋 Middlewares (%d):\n", len(r.middlewares))
	for i, middleware := range r.middlewares {
		fmt.Printf("  %d. %T at %p\n", i+1, middleware, middleware)
	}

	fmt.Printf("\n🎯 Handlers (%d):\n", len(r.handlers))
	i := 1
	for path, handler := range r.handlers {
		fmt.Printf("  %d. Path: %-20s Handler: %T at %p\n", i, path, handler, handler)
		i++
	}

	if r.notFound != nil {
		fmt.Printf("\n❌ 404 Handler: %T at %p\n", r.notFound, r.notFound)
	} else {
		fmt.Printf("\n❌ 404 Handler: not set\n")
	}

	if r.recoverr != nil {
		fmt.Printf("🔄 Recover Handler: %T at %p\n", r.recoverr, r.recoverr)
	} else {
		fmt.Printf("🔄 Recover Handler: not set\n")
	}

	fmt.Println("=======================================")
}

func (r *Router) Start() {
	r.printHandlersInfo()

	fmt.Printf("\n🚀 Bot started! Listening for updates...\n")
	fmt.Printf("📝 Available commands: %s\n", strings.Join(r.getCommandList(), ", "))
	fmt.Println()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := r.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			fmt.Printf("📨 Received message: %s (from: %s %s)\n",
				update.Message.Text,
				update.Message.From.FirstName,
				update.Message.From.LastName)

			data := map[string]interface{}{
				"chat_id": update.Message.Chat.ID,
				"path":    update.Message.Text,
				"params":  "",
			}

			r.Handle(update.Message.Text, data)
		} else if update.CallbackQuery != nil {
			fmt.Println(update.CallbackQuery.Data)

			parts := strings.Split(update.CallbackQuery.Data, ":")
			command := parts[0]
			id, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				panic(err)
			}

			param := ""
			if len(parts) > 2 {
				param = parts[2]
			}

			data := map[string]interface{}{
				"chat_id": id,
				"path":    command,
				"param":   param,
			}

			fmt.Printf("📨 Received message: %s\n", command)

			r.Handle(command, data)
		}
	}

	select {}
}

func (r *Router) getCommandList() []string {
	commands := make([]string, 0, len(r.handlers))
	for path := range r.handlers {
		commands = append(commands, path)
	}
	return commands
}
