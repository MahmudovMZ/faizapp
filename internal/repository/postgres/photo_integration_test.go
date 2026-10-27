//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPhotoRepository(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := NewFaizAppRepo(pool)

	const tgID int64 = 999999994

	var userID string
	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, status)
		VALUES ($1, $2, 'approved')
		RETURNING id
		`,
		tgID,
		"Photo Integration User",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	defer func() {
		_, _ = pool.Exec(
			ctx,
			`DELETE FROM photos WHERE user_id = $1`,
			userID,
		)

		_, _ = pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	}()

	receivedAt := time.Now().UTC().Truncate(time.Microsecond)

	photo := &models.Photo{
		UserID:            userID,
		TelegramUserID:    tgID,
		TelegramMessageID: 1001,
		Sha256Hash:        "integration-test-hash",
		Caption:           "test photo",
		ReceivedAt:        receivedAt,
	}

	if err := repo.CreatePhoto(ctx, photo); err != nil {
		t.Fatalf("create photo: %v", err)
	}

	countByUser, err := repo.CountPhotosByUser(ctx, userID)
	if err != nil {
		t.Fatalf("count photos by user: %v", err)
	}

	if countByUser != 1 {
		t.Fatalf("count by user = %d, want 1", countByUser)
	}

	from := receivedAt.Add(-time.Minute)
	to := receivedAt.Add(time.Minute)

	countByDate, err := repo.CountPhotosByDate(ctx, from, to)
	if err != nil {
		t.Fatalf("count photos by date: %v", err)
	}

	if countByDate != 1 {
		t.Fatalf("count by date = %d, want 1", countByDate)
	}
}

func TestPhotoRepositoryDuplicateHash(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := NewFaizAppRepo(pool)

	const tgID int64 = 999999995

	var userID string
	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, status)
		VALUES ($1, $2, 'approved')
		RETURNING id
		`,
		tgID,
		"Duplicate Photo Test User",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	defer func() {
		_, _ = pool.Exec(
			ctx,
			`DELETE FROM photos WHERE user_id = $1`,
			userID,
		)

		_, _ = pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	}()

	receivedAt := time.Now().UTC().Truncate(time.Microsecond)

	firstPhoto := &models.Photo{
		UserID:            userID,
		TelegramUserID:    tgID,
		TelegramMessageID: 2001,
		Sha256Hash:        "duplicate-integration-test-hash",
		Caption:           "first photo",
		ReceivedAt:        receivedAt,
	}

	if err := repo.CreatePhoto(ctx, firstPhoto); err != nil {
		t.Fatalf("create first photo: %v", err)
	}

	secondPhoto := &models.Photo{
		UserID:            userID,
		TelegramUserID:    tgID,
		TelegramMessageID: 2002,
		Sha256Hash:        firstPhoto.Sha256Hash,
		Caption:           "duplicate photo",
		ReceivedAt:        receivedAt,
	}

	err = repo.CreatePhoto(ctx, secondPhoto)
	if !errors.Is(err, ErrDuplicatePhoto) {
		t.Fatalf(
			"error = %v, want ErrDuplicatePhoto",
			err,
		)
	}

	count, err := repo.CountPhotosByUser(ctx, userID)
	if err != nil {
		t.Fatalf("count photos by user: %v", err)
	}

	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
