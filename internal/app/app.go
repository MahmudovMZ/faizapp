package app

import (
	"log"

	"github.com/MahmudovMZ/faizapp/internal/config"
	"github.com/MahmudovMZ/faizapp/internal/polling"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(cfg config.Config, pool *pgxpool.Pool) {
	log.Println("Starting app")

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
