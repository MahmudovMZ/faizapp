package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5"
)

func (r *FaizAppRepo) EnqueuePhotoJob(ctx context.Context, job *models.PhotoJob) error {
	log.Println("[REPOSITORY] Enqueue photo job")

	query := `INSERT INTO photo_jobs (chat_id,telegram_message_id, telegram_user_id, sender_name, file_id, caption, media_group_id, received_at) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7, ''),$8) ON CONFLICT (chat_id, telegram_message_id) DO NOTHING`

	_, err := r.Pool.Exec(ctx, query, job.ChatID, job.TelegramMessageID, job.TelegramUserID, job.SenderName, job.FileID, job.Caption, job.MediaGroupID, job.ReceivedAt)
	if err != nil {
		return fmt.Errorf("error inserting photo job: %w", err)
	}
	return nil
}

func (r *FaizAppRepo) ClaimPhotoJob(ctx context.Context, lockFor time.Duration) (*models.PhotoJob, error) {
	log.Println("[REPOSITORY] Claim photo job")

	query := `UPDATE photo_jobs SET status = $1, locked_until = NOW() + $2 * INTERVAL '1 second',
              attempts = attempts +1, updated_at = NOW() WHERE id = (SELECT id FROM photo_jobs
              WHERE (status = 'pending' AND next_attempt_at <= NOW()) OR (status = $3 AND locked_until < NOW())
              ORDER BY next_attempt_at LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING id, chat_id, telegram_message_id, telegram_user_id, sender_name,
              file_id, caption, COALESCE(media_group_id, ''), received_at,
              status, attempts, next_attempt_at, locked_until, last_error, created_at, updated_at`
	var job models.PhotoJob
	err := r.Pool.QueryRow(ctx, query, models.PhotoJobProcessing, lockFor.Seconds(), models.PhotoJobProcessing).Scan(
		&job.ID,
		&job.ChatID,
		&job.TelegramMessageID,
		&job.TelegramUserID,
		&job.SenderName,
		&job.FileID,
		&job.Caption,
		&job.MediaGroupID,
		&job.ReceivedAt,
		&job.Status,
		&job.Attempts,
		&job.NextAttemptAt,
		&job.LockedUntil,
		&job.LastError,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim photo job: %w", err)
	}
	return &job, nil
}

func (r *FaizAppRepo) MarkPhotoJobDone(ctx context.Context, id int64) error {
	log.Println("[REPOSITORY] Mark photo job done")

	query := `UPDATE photo_jobs SET status = $1, locked_until = NULL, updated_at = NOW() WHERE id = $2`

	stat, err := r.Pool.Exec(ctx, query, models.PhotoJobDone, id)
	if err != nil {
		return fmt.Errorf("mark photo job done: %w", err)
	}
	if stat.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
func (r *FaizAppRepo) MarkPhotoJobDead(ctx context.Context, id int64, errTxt string) error {
	log.Println("[REPOSITORY] Mark photo job dead")

	query := `UPDATE photo_jobs SET status = $1, locked_until = NULL, updated_at = NOW(), last_error = $2 WHERE id = $3`

	stat, err := r.Pool.Exec(ctx, query, models.PhotoJobDead, errTxt, id)
	if err != nil {
		return fmt.Errorf("mark photo job dead: %w", err)
	}
	if stat.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *FaizAppRepo) RetryPhotoJob(ctx context.Context, id int64, retryAfter time.Duration, errTxt string) error {
	log.Println("[REPOSITORY] Retry photo job")

	query := `UPDATE photo_jobs SET status = $1,locked_until = NULL, updated_at = NOW(),last_error = $2, next_attempt_at = NOW() + $3 * INTERVAL '1 second' WHERE id = $4`

	stat, err := r.Pool.Exec(ctx, query, models.PhotoJobPending, errTxt, retryAfter.Seconds(), id)
	if err != nil {
		return fmt.Errorf("retry photo job: %w", err)
	}
	if stat.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
