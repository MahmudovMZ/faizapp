package polling

import (
	"log"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/service"
	"github.com/MahmudovMZ/faizapp/internal/transport/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartPolling(bot *tgbotapi.BotAPI, userService *service.UserService, photoService *service.PhotoService, job *service.PhotoJobService, botAdmin int64, photoTimeout time.Duration, photoWorkers int) {

	log.Println("Бот запущен!")

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)
	handler := telegram.NewBotHandler(bot, userService, photoService, job, botAdmin, photoTimeout)

	for i := 0; i < photoWorkers; i++ {
		go func(id int) {
			handler.RunPhotoWorker(id)
		}(i + 1)
	}

	for update := range updates {
		if update.Message != nil && len(update.Message.Photo) > 0 {
			handler.EnqueuePhoto(update.Message)
			continue
		}
		handler.BotHandler(update)
	}

}
