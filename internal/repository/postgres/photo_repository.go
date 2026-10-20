package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
)

func (r *FaizAppRepo) CreatePhoto(
	ctx context.Context,
	photo *models.Photo,
) error {
	log.Println("[REPOSITORY] creating photo")

	query := `
		INSERT INTO photos (
			user_id,
			telegram_user_id,
			telegram_message_id,
			sha256_hash,
			caption,
			received_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.Pool.Exec(
		ctx,
		query,
		photo.UserID,
		photo.TelegramUserID,
		photo.TelegramMessageID,
		photo.Sha256Hash,
		photo.Caption,
		photo.ReceivedAt,
	)
	if err != nil {
		return fmt.Errorf("create photo: %w", err)
	}

	return nil
}

func (r *FaizAppRepo) CountPhotosByUser(ctx context.Context, userID string) (int, error) {
	log.Println("[REPOSITORY] counting photos by user")

	query := `SELECT COUNT(*) FROM photos WHERE user_id = $1`

	var count int
	row := r.Pool.QueryRow(ctx, query, userID)
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count photos by user: %w", err)
	}
	return count, nil
}

func (r *FaizAppRepo) CountPhotosByDate(ctx context.Context, from time.Time, to time.Time) (int, error) {
	log.Println("[REPOSITORY] counting photos by date")

	query := `SELECT COUNT(*) FROM photos WHERE received_at >= $1 AND received_at < $2`
	var count int
	row := r.Pool.QueryRow(ctx, query, from, to)
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count photos by date: %w", err)
	}
	return count, nil
}
