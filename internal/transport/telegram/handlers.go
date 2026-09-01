package telegram

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var botState = make(map[int64]string)
var userState = make(map[int64]string)
var userData = make(map[int64]map[string]string)
var bot *tgbotapi.BotAPI

func BotHandler(bot2 *tgbotapi.BotAPI, update tgbotapi.Update) {
	bot = bot2

	if update.Message == nil {
		return
	}

	chatId := update.Message.Chat.ID
	text := update.Message.Text

	stage := botState[chatId]

	switch stage {
	case "":
		if text == "/start" {
			send(chatId, "You are contacting th FaizApp now")
			send(chatId, "How I can assist You?")
		}
	}

}

func send(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("Error sending message: ", err)
		return
	}
}
