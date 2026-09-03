package app

import (
	"log"

	"github.com/MahmudovMZ/faizapp/internal/config"
	"github.com/MahmudovMZ/faizapp/internal/polling"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Run(cfg config.Config) {
	log.Println("Starting app")

	// TODO: Db connection

	bot, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
	if err != nil {
		log.Fatal("[RUNNING] TGBOTAPI.NewBotAPI: " + err.Error())
	}
	bot.Debug = true

	switch cfg.Bot.BotMode {
	case "polling":
		polling.StartPolling(bot)
	}
}
