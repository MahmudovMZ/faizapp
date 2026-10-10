package polling

import (
	"log"

	"github.com/MahmudovMZ/faizapp/internal/service"
	"github.com/MahmudovMZ/faizapp/internal/transport/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartPolling(bot *tgbotapi.BotAPI, userService *service.UserService, botAdmin int64) {

	log.Println("Бот запущен!")

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)
	handler := telegram.NewBotHandler(bot, userService, botAdmin)

	for update := range updates {
		handler.BotHandler(update)
	}

}
