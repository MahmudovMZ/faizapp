package app

import (
	"log"
	"os"

	"github.com/MahmudovMZ/faizapp/internal/polling"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func Run() {
	log.Println("Starting app")
	//err := config.ReadConfig("internal/config/config.json")
	////if err != nil {
	////	log.Fatal("Could not read config files", err)
	////}

	// TODO: Db connection
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("[RUNNING] .env file was not found")
	}
	var tgToken = os.Getenv("TGBOTAPI_TOKEN")
	var botMode = os.Getenv("BOT_MODE")

	if tgToken == "" || botMode == "" {
		log.Fatal("[RUNNING] TGBOTAPI_TOKEN or BOT_MODE environment variables not set")
	}

	bot, err := tgbotapi.NewBotAPI(tgToken)
	if err != nil {
		log.Fatal("[RUNNING] TGBOTAPI.NewBotAPI: " + err.Error())
	}
	bot.Debug = true

	switch botMode {
	case "polling":
		polling.StartPolling(bot)
	}
}
