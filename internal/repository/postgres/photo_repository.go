package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrDuplicatePhoto = errors.New(
	"photo with this SHA-256 hash already exists",
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
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_photos_sha256_hash" {
			return fmt.Errorf("create photo: %w", ErrDuplicatePhoto)
		}
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

	query := `SELECT COUNT(*)
				FROM photos p
				JOIN users u ON u.id = p.user_id
				WHERE u.role IN ($3, $4)
				AND p.received_at >= $1
				AND p.received_at < $2`
	var count int
	row := r.Pool.QueryRow(ctx, query, from, to, models.RoleSalesRep, models.RoleSupervisor)
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count photos by date: %w", err)
	}
	return count, nil
}
