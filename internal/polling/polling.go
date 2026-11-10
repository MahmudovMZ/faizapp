package polling

import (
	"log"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/service"
	"github.com/MahmudovMZ/faizapp/internal/transport/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartPolling(bot *tgbotapi.BotAPI, userService *service.UserService, photoService *service.PhotoService, botAdmin int64, photoTimeout time.Duration, photoWorkers int) {

	log.Println("Бот запущен!")

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)
	handler := telegram.NewBotHandler(bot, userService, photoService, botAdmin, photoTimeout)

	photos := make(chan tgbotapi.Update, 100)

	for i := 0; i < photoWorkers; i++ {
		go func(id int) {
			log.Printf("[WORKER %d] started", id) // один раз при запуске
			for update := range photos {
				log.Printf("[WORKER %d] processing photo from %d", id, update.Message.From.ID)
				handler.BotHandler(update)
			}
		}(i + 1)
	}

	for update := range updates {
		if update.Message != nil && len(update.Message.Photo) > 0 {
			photos <- update
			continue
		}
		handler.BotHandler(update)
	}

}
