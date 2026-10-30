package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handlePhotoMessage(ctx context.Context, msg *tgbotapi.Message) (service.SavePhotoResult, error) {
	if msg == nil || msg.From == nil {
		return service.SavePhotoResult{}, errors.New("no message from user")
	}
	tgUserID := msg.From.ID
	tgMessageID := msg.MessageID
	caption := msg.Caption

	if len(msg.Photo) == 0 {
		return service.SavePhotoResult{}, errors.New("[HANDLER] photo not found")
	}
	photoSize := msg.Photo[len(msg.Photo)-1]
	receivedAt := time.Unix(int64(msg.Date), 0)
	fileID := photoSize.FileID

	data, err := h.downloadTelegramFile(ctx, fileID)
	if err != nil {
		return service.SavePhotoResult{}, fmt.Errorf("[HANDLER] download telegram file failed: %w", err)
	}

	fullName := fmt.Sprintf("%s %s", strings.TrimSpace(msg.From.FirstName), strings.TrimSpace(msg.From.LastName))

	result, err := h.PhotoService.SavePhoto(ctx, tgUserID, tgMessageID, data, caption, receivedAt, strings.TrimSpace(fullName))
	if err != nil {
		return service.SavePhotoResult{}, fmt.Errorf("[HANDLER] save photo failed: %w", err)
	}
	return result, nil
}

func (h *Handler) downloadTelegramFile(ctx context.Context, fileID string) ([]byte, error) {
	file, err := h.TelegramBot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("[HANDLER] telegram get file %s error: %w", fileID, err)
	}
	url := file.Link(h.TelegramBot.Token)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[HANDLER] create request: %w", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("[HANDLER] download file %s error: %w", fileID, err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Printf(
				"[HANDLER] close telegram file body error: %v",
				err,
			)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[HANDLER] download file %s error: %s", fileID, response.Status)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("[HANDLER] read file %s error: %w", fileID, err)
	}
	return data, nil
}
