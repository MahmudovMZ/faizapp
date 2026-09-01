package polling

import (
	"log"

	"github.com/MahmudovMZ/faizapp/internal/transport/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartPolling(bot *tgbotapi.BotAPI) {

	log.Println("Бот запущен!")

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		telegram.BotHandler(bot, update)
	}

}
