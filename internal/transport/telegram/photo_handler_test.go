package telegram

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestHandlePhotoMessageNilMessage(t *testing.T) {
	handler := &Handler{}

	_, err := handler.handlePhotoMessage(
		context.Background(),
		nil,
	)

	if err == nil {
		t.Fatal("expected error for nil message, got nil")
	}
}

func TestHandlePhotoMessageWithoutSender(t *testing.T) {
	handler := &Handler{}

	message := &tgbotapi.Message{
		Photo: []tgbotapi.PhotoSize{
			{
				FileID: "test-file-id",
			},
		},
	}

	_, err := handler.handlePhotoMessage(
		context.Background(),
		message,
	)

	if err == nil {
		t.Fatal("expected error for message without sender, got nil")
	}
}

func TestHandlePhotoMessageWithoutPhoto(t *testing.T) {
	handler := &Handler{}

	message := &tgbotapi.Message{
		From: &tgbotapi.User{
			ID: 123456789,
		},
	}

	_, err := handler.handlePhotoMessage(
		context.Background(),
		message,
	)

	if err == nil {
		t.Fatal("expected error for message without photo, got nil")
	}
}
