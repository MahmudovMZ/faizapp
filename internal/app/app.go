package app

import (
	"log"

	"github.com/MahmudovMZ/faizapp/internal/config"
	"github.com/MahmudovMZ/faizapp/internal/polling"
	"github.com/MahmudovMZ/faizapp/internal/repository/postgres"
	"github.com/MahmudovMZ/faizapp/internal/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(cfg config.Config, pool *pgxpool.Pool) error {
	log.Println("Starting app")

	repo := postgres.NewFaizAppRepo(pool)
	userService := service.NewUserService(repo)

	bot, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
	if err != nil {
		return err
	}
	//bot.Debug = true

	switch cfg.Bot.BotMode {
	case "polling":
		polling.StartPolling(bot, userService)
	}

	return nil
}
