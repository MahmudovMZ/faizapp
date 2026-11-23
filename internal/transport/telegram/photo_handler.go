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

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) processPhotoJob(ctx context.Context, job *models.PhotoJob) error {
	data, err := h.downloadTelegramFile(ctx, job.FileID)
	if err != nil {
		return fmt.Errorf("[HANDLER] download telegram file failed: %w", err)
	}

	result, err := h.PhotoService.SavePhoto(ctx, job.TelegramUserID, job.TelegramMessageID, data, job.Caption, job.ReceivedAt, job.SenderName, job.MediaGroupID)
	if err != nil {
		if errors.Is(err, service.ErrDuplicatePhoto) {
			h.replyTo(job.ChatID, job.TelegramMessageID, rejectPhotoText)
			h.sendRegistrationLinkOnce(ctx, job)
		}
		return fmt.Errorf("[HANDLER] save photo failed: %w", err)
	}

	h.sendRegistrationLinkOnce(ctx, job)
	log.Printf("[TELEGRAM] photo saved: user=%d count=%d counted=%v", job.TelegramUserID, result.Count, result.Counted)
	return nil
}

func (h *Handler) EnqueuePhoto(msg *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if msg == nil || msg.From == nil {
		log.Println("[TELEGRAM] msg.From is nil")
		return
	}

	if len(msg.Photo) == 0 {
		log.Println("[TELEGRAM] enqueue photo: message has no photo")
		return
	}
	fullName := fmt.Sprintf("%s %s", strings.TrimSpace(msg.From.FirstName), strings.TrimSpace(msg.From.LastName))
	photoSize := msg.Photo[len(msg.Photo)-1]
	receivedAt := time.Unix(int64(msg.Date), 0)
	fileID := photoSize.FileID

	photoJob := &models.PhotoJob{
		ChatID:            msg.Chat.ID,
		TelegramMessageID: msg.MessageID,
		TelegramUserID:    msg.From.ID,
		SenderName:        strings.TrimSpace(fullName),
		FileID:            fileID,
		Caption:           msg.Caption,
		MediaGroupID:      msg.MediaGroupID,
		ReceivedAt:        receivedAt,
	}

	err := h.JobService.EnqueueJob(ctx, photoJob)
	if err != nil {
		h.replyTo(msg.Chat.ID, msg.MessageID, failPhotoText)
		log.Printf("[HANDLER] enqueue photo failed: %v", err)
	}
}

const emptyQueueDelay = 3 * time.Second

func (h *Handler) RunPhotoWorker(workerID int) {
	log.Printf("[HANDLER] starting worker %d", workerID)
	for {
		ctx1, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		job, err := h.JobService.NextJob(ctx1)
		cancel()
		if err != nil {
			log.Printf("[HANDLER] next job failed: %v", err)
			time.Sleep(emptyQueueDelay)
			continue
		}
		if job == nil {
			time.Sleep(emptyQueueDelay)
			continue
		}

		log.Printf("[HANDLER] worker %d received job %d from user %d", workerID, job.ID, job.TelegramUserID)
		ctx2, cancel := context.WithTimeout(context.Background(), h.photoTimeout)
		processErr := h.processPhotoJob(ctx2, job)
		cancel()
		if processErr != nil {
			log.Printf("[HANDLER] process photo job failed: %v", processErr)
		}

		log.Printf("[HANDLER] worker %d finished job %d from user %d", workerID, job.ID, job.TelegramUserID)
		ctx3, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = h.JobService.FinishJob(ctx3, job, processErr)
		cancel()
		if err != nil {
			log.Printf("[HANDLER] finish job failed: %v", err)
		}
	}

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

func (h *Handler) sendRegistrationLinkOnce(ctx context.Context, job *models.PhotoJob) {
	claimed, err := h.PhotoService.ClaimRegistrationLink(ctx, job.TelegramUserID)
	if err != nil {
		log.Printf("[HANDLER] send registration link once failed: %v", err)
		return
	}

	if claimed {
		h.mentionNewUser(job.ChatID, job.TelegramMessageID, job.TelegramUserID, job.SenderName)
	}
}
