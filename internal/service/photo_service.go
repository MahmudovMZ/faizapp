package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/repository/postgres"
)

type PhotoService struct {
	repo postgres.Repository
}

var ErrDuplicatePhoto = errors.New("photo already exists")

func NewPhotoService(repo postgres.Repository) *PhotoService {
	return &PhotoService{repo: repo}
}

func (s *PhotoService) SavePhoto(ctx context.Context, tgId int64, tgMsgId int, data []byte, caption string, receivedAt time.Time) (int, error) {
	user, err := s.repo.GetUserByTgID(ctx, tgId)
	if err != nil {
		return 0, fmt.Errorf("[SERVICE] get user by telegram ID: %w", err)
	}
	if user == nil {
		return 0, errors.New("[SERVICE] user not found")
	}

	hash := CalcSHA256(data)
	photo := &models.Photo{
		UserID:            user.ID,
		TelegramUserID:    tgId,
		TelegramMessageID: tgMsgId,
		Sha256Hash:        hash,
		Caption:           caption,
		ReceivedAt:        receivedAt,
	}

	err = s.repo.CreatePhoto(ctx, photo)
	if err != nil {
		if errors.Is(err, postgres.ErrDuplicatePhoto) {
			return 0, fmt.Errorf("[SERVICE] photo already exists: %w", ErrDuplicatePhoto)
		}
		return 0, fmt.Errorf("[SERVICE] saving photo: %w", err)
	}

	num, err := s.repo.CountPhotosByUser(ctx, user.ID)
	if err != nil {
		return 0, fmt.Errorf("[SERVICE] count photo: %w", err)
	}

	return num, nil
}
