package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// The handler has no JobService here: if a guard let the message through,
// EnqueuePhoto would panic on the nil service, so returning quietly means
// the message was rejected before any queue or Telegram call.

func TestEnqueuePhotoNilMessage(t *testing.T) {
	handler := &Handler{}

	handler.EnqueuePhoto(nil)
}

func TestEnqueuePhotoWithoutSender(t *testing.T) {
	handler := &Handler{}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: -100},
		Photo: []tgbotapi.PhotoSize{
			{FileID: "test-file-id"},
		},
	}

	handler.EnqueuePhoto(message)
}

func TestEnqueuePhotoWithoutPhoto(t *testing.T) {
	handler := &Handler{}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: -100},
		From: &tgbotapi.User{ID: 123456789},
	}

	handler.EnqueuePhoto(message)
}
