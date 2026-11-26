//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
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
		INSERT INTO users (tg_id, full_name, status, role)
		VALUES ($1, $2, 'approved', $3)
		RETURNING id
		`,
		tgID,
		"Photo Integration User",
		models.RoleSalesRep,
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

func TestCreatePhotoMediaGroupID(t *testing.T) {
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

	tgID := randomTestTgID()

	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM photos WHERE telegram_user_id = $1`, tgID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE tg_id = $1`, tgID)
	}()

	user, err := repo.CreatePendingUser(ctx, tgID, "Media Group User")
	if err != nil {
		t.Fatalf("create pending user: %v", err)
	}

	albumPhoto := &models.Photo{
		UserID:            user.ID,
		TelegramUserID:    tgID,
		TelegramMessageID: 4001,
		MediaGroupID:      "album-test-1",
		Sha256Hash:        fmt.Sprintf("album-photo-%d", tgID),
		ReceivedAt:        time.Now().UTC(),
	}

	singlePhoto := &models.Photo{
		UserID:            user.ID,
		TelegramUserID:    tgID,
		TelegramMessageID: 4002,
		Sha256Hash:        fmt.Sprintf("single-photo-%d", tgID),
		ReceivedAt:        time.Now().UTC(),
	}

	for _, photo := range []*models.Photo{albumPhoto, singlePhoto} {
		if err := repo.CreatePhoto(ctx, photo); err != nil {
			t.Fatalf("create photo %d: %v", photo.TelegramMessageID, err)
		}
	}

	var groupID *string

	err = pool.QueryRow(
		ctx,
		`SELECT media_group_id FROM photos WHERE telegram_user_id = $1 AND telegram_message_id = $2`,
		tgID,
		4001,
	).Scan(&groupID)
	if err != nil {
		t.Fatalf("read album photo: %v", err)
	}
	if groupID == nil || *groupID != "album-test-1" {
		t.Fatalf("album media_group_id = %v, want album-test-1", groupID)
	}

	err = pool.QueryRow(
		ctx,
		`SELECT media_group_id FROM photos WHERE telegram_user_id = $1 AND telegram_message_id = $2`,
		tgID,
		4002,
	).Scan(&groupID)
	if err != nil {
		t.Fatalf("read single photo: %v", err)
	}
	if groupID != nil {
		t.Fatalf("single photo media_group_id = %q, want NULL", *groupID)
	}
}

func TestPhotoExistsForMessage(t *testing.T) {
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

	repo := NewFaizAppRepo(pool)

	tgID := randomTestTgID()

	var userID string
	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, status)
		VALUES ($1, $2, 'approved')
		RETURNING id
		`,
		tgID,
		"Photo Exists Test User",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM photos WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	}()

	photo := &models.Photo{
		UserID:            userID,
		TelegramUserID:    tgID,
		TelegramMessageID: 3001,
		Sha256Hash:        fmt.Sprintf("photo-exists-test-hash-%d", tgID),
		ReceivedAt:        time.Now().UTC(),
	}

	if err := repo.CreatePhoto(ctx, photo); err != nil {
		t.Fatalf("create photo: %v", err)
	}

	tests := []struct {
		name    string
		tgID    int64
		tgMsgID int
		want    bool
	}{
		{name: "same sender and message", tgID: tgID, tgMsgID: 3001, want: true},
		{name: "same sender, other message", tgID: tgID, tgMsgID: 3002, want: false},
		{name: "other sender, same message", tgID: tgID + 1, tgMsgID: 3001, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := repo.PhotoExistsForMessage(ctx, tt.tgID, tt.tgMsgID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if exists != tt.want {
				t.Fatalf("exists = %v, want %v", exists, tt.want)
			}
		})
	}
}
