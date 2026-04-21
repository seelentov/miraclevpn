package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"miraclevpn/internal/config/db"
	"miraclevpn/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: broadcast <message>")
		os.Exit(1)
	}
	message := strings.Join(args, " ")

	token := os.Getenv("TG_HANDLER_TOKEN")
	if token == "" {
		log.Fatal("TG_HANDLER_TOKEN not set")
	}

	gormDB, err := db.NewConnFromEnv()
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	var users []models.User
	if err := gormDB.Where("active = ?", true).Find(&users).Error; err != nil {
		log.Fatalf("fetch users: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("bot init: %v", err)
	}

	ok, failed := 0, 0
	for _, u := range users {
		if strings.HasPrefix(u.ID, "web-") {
			continue
		}
		msg := tgbotapi.NewMessage(0, message)
		msg.ChatID, _ = parseChatID(u.ID)
		msg.ParseMode = "Markdown"
		if _, err := bot.Send(msg); err != nil {
			log.Printf("failed %s: %v", u.ID, err)
			failed++
		} else {
			ok++
		}
		time.Sleep(50 * time.Millisecond) // stay within Telegram rate limits
	}

	fmt.Printf("Done: %d sent, %d failed\n", ok, failed)
}

func parseChatID(id string) (int64, error) {
	var n int64
	_, err := fmt.Sscan(id, &n)
	return n, err
}
