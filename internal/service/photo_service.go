package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/repository/postgres"
	"github.com/jackc/pgx/v5"
)

type PhotoService struct {
	repo postgres.Repository
}

var ErrDuplicatePhoto = errors.New("photo already exists")

type SavePhotoResult struct {
	Count     int
	Counted   bool
	NewSender bool
}

func NewPhotoService(repo postgres.Repository) *PhotoService {
	return &PhotoService{repo: repo}
}

func (s *PhotoService) SavePhoto(ctx context.Context, tgId int64, tgMsgId int, data []byte, caption string, receivedAt time.Time, fullName string) (SavePhotoResult, error) {
	isNew := false
	user, err := s.repo.GetUserByTgID(ctx, tgId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			isNew = true
			user, err = s.repo.CreatePendingUser(ctx, tgId, fullName)
			if err != nil {
				return SavePhotoResult{}, fmt.Errorf("[SERVICE] create pending user: %w", err)
			}
		} else {
			return SavePhotoResult{}, fmt.Errorf("[SERVICE] get user by telegram ID: %w", err)
		}

	}
	if user == nil {
		return SavePhotoResult{}, errors.New("[SERVICE] user not found")
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
			return SavePhotoResult{}, fmt.Errorf("[SERVICE] photo already exists: %w", ErrDuplicatePhoto)
		}
		return SavePhotoResult{}, fmt.Errorf("[SERVICE] saving photo: %w", err)
	}

	num, err := s.repo.CountPhotosByUser(ctx, user.ID)
	if err != nil {
		return SavePhotoResult{}, fmt.Errorf("[SERVICE] count photo: %w", err)
	}

	return SavePhotoResult{
		Count:     num,
		Counted:   isRoleCounted(user.Role),
		NewSender: isNew,
	}, nil
}

func isRoleCounted(role *string) bool {
	if role == nil {
		return false
	}
	return *role == models.RoleSalesRep || *role == models.RoleSupervisor
}
