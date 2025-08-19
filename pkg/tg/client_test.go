package tg

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

var client *TgClient

func setup() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal(err)
	}

	tgToken := os.Getenv("TG_TOKEN")
	tgName := os.Getenv("TG_NAME")

	client = NewTgClient(tgToken, tgName)
}

func teardown() {

}

func TestMain(m *testing.M) {
	setup()
	exitVal := m.Run()
	teardown()
	os.Exit(exitVal)
}

func TestClient_SendMessage(t *testing.T) {
	err := client.SendMessage("816233444", "test")
	if err != nil {
		t.Error(err)
	}
}
