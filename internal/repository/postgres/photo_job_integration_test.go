//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEnqueuePhotoJob(t *testing.T) {
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

	// A random chat ID keeps this test independent from other rows.
	chatID := -randomTestTgID()

	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM photo_jobs WHERE chat_id = $1`, chatID)
	}()

	receivedAt := time.Now().UTC().Truncate(time.Microsecond)

	albumJob := &models.PhotoJob{
		ChatID:            chatID,
		TelegramMessageID: 5001,
		TelegramUserID:    42,
		SenderName:        "Queue Test User",
		FileID:            "file-album",
		Caption:           "before",
		MediaGroupID:      "album-queue-1",
		ReceivedAt:        receivedAt,
	}

	singleJob := &models.PhotoJob{
		ChatID:            chatID,
		TelegramMessageID: 5002,
		TelegramUserID:    42,
		FileID:            "file-single",
		ReceivedAt:        receivedAt,
	}

	for _, job := range []*models.PhotoJob{albumJob, singleJob} {
		if err := repo.EnqueuePhotoJob(ctx, job); err != nil {
			t.Fatalf("enqueue message %d: %v", job.TelegramMessageID, err)
		}
	}

	// The same Telegram message delivered twice must not create a second job.
	if err := repo.EnqueuePhotoJob(ctx, albumJob); err != nil {
		t.Fatalf("enqueue duplicate message: %v", err)
	}

	var rows int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM photo_jobs WHERE chat_id = $1`, chatID).Scan(&rows)
	if err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if rows != 2 {
		t.Fatalf("jobs = %d, want 2", rows)
	}

	var (
		status       string
		attempts     int
		fileID       string
		senderName   string
		caption      string
		mediaGroupID *string
		lockedUntil  *time.Time
		savedAt      time.Time
	)

	err = pool.QueryRow(
		ctx,
		`
		SELECT status, attempts, file_id, sender_name, caption,
		       media_group_id, locked_until, received_at
		FROM photo_jobs
		WHERE chat_id = $1 AND telegram_message_id = $2
		`,
		chatID,
		5001,
	).Scan(&status, &attempts, &fileID, &senderName, &caption, &mediaGroupID, &lockedUntil, &savedAt)
	if err != nil {
		t.Fatalf("read album job: %v", err)
	}

	if status != models.PhotoJobPending {
		t.Fatalf("status = %q, want %q", status, models.PhotoJobPending)
	}
	if attempts != 0 {
		t.Fatalf("attempts = %d, want 0", attempts)
	}
	if fileID != "file-album" || senderName != "Queue Test User" || caption != "before" {
		t.Fatalf("stored (%q, %q, %q), want (file-album, Queue Test User, before)", fileID, senderName, caption)
	}
	if mediaGroupID == nil || *mediaGroupID != "album-queue-1" {
		t.Fatalf("media_group_id = %v, want album-queue-1", mediaGroupID)
	}
	if lockedUntil != nil {
		t.Fatalf("locked_until = %v, want NULL", *lockedUntil)
	}
	if !savedAt.Equal(receivedAt) {
		t.Fatalf("received_at = %v, want %v", savedAt, receivedAt)
	}

	err = pool.QueryRow(
		ctx,
		`SELECT media_group_id FROM photo_jobs WHERE chat_id = $1 AND telegram_message_id = $2`,
		chatID,
		5002,
	).Scan(&mediaGroupID)
	if err != nil {
		t.Fatalf("read single job: %v", err)
	}
	if mediaGroupID != nil {
		t.Fatalf("single job media_group_id = %q, want NULL", *mediaGroupID)
	}
}

// newPhotoJobTestRepo connects to the test database and removes every job
// of the given chat when the test ends.
func newPhotoJobTestRepo(t *testing.T, chatID int64) (*FaizAppRepo, *pgxpool.Pool) {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping database: %v", err)
	}

	// Registered after cancel, so it runs before it (t.Cleanup is LIFO)
	// and the pool is still open when the rows are deleted.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM photo_jobs WHERE chat_id = $1`, chatID)
		pool.Close()
	})

	return NewFaizAppRepo(pool), pool
}

func enqueueTestJob(t *testing.T, repo *FaizAppRepo, chatID int64, messageID int) {
	t.Helper()

	job := &models.PhotoJob{
		ChatID:            chatID,
		TelegramMessageID: messageID,
		TelegramUserID:    42,
		FileID:            "file-claim",
		ReceivedAt:        time.Now().UTC(),
	}

	if err := repo.EnqueuePhotoJob(context.Background(), job); err != nil {
		t.Fatalf("enqueue message %d: %v", messageID, err)
	}
}

func TestClaimPhotoJob(t *testing.T) {
	chatID := -randomTestTgID()
	repo, pool := newPhotoJobTestRepo(t, chatID)
	ctx := context.Background()

	requireNoOtherReadyJobs(t, pool, chatID)

	job, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim from empty queue: %v", err)
	}
	if job != nil {
		t.Fatalf("claim from empty queue = job %d, want nil", job.ID)
	}

	enqueueTestJob(t, repo, chatID, 6001)

	before := time.Now()

	job, err = repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if job == nil {
		t.Fatal("claim = nil, want a job")
	}

	if job.ChatID != chatID || job.TelegramMessageID != 6001 || job.FileID != "file-claim" {
		t.Fatalf("claimed wrong job: chat %d message %d file %q", job.ChatID, job.TelegramMessageID, job.FileID)
	}
	if job.Status != models.PhotoJobProcessing {
		t.Fatalf("status = %q, want %q", job.Status, models.PhotoJobProcessing)
	}
	if job.Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", job.Attempts)
	}
	if job.MediaGroupID != "" {
		t.Fatalf("media group id = %q, want empty", job.MediaGroupID)
	}
	if job.LockedUntil == nil {
		t.Fatal("locked_until = nil, want a time")
	}

	// The lock must be about one minute ahead, not seconds and not years.
	lockedFor := job.LockedUntil.Sub(before)
	if lockedFor < 50*time.Second || lockedFor > 70*time.Second {
		t.Fatalf("locked for %v, want about 1m", lockedFor)
	}

	again, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim locked job: %v", err)
	}
	if again != nil {
		t.Fatalf("claimed job %d while it is locked, want nil", again.ID)
	}
}

func TestClaimPhotoJobAbandonedAndFuture(t *testing.T) {
	chatID := -randomTestTgID()
	repo, pool := newPhotoJobTestRepo(t, chatID)
	ctx := context.Background()

	requireNoOtherReadyJobs(t, pool, chatID)

	enqueueTestJob(t, repo, chatID, 6101)
	enqueueTestJob(t, repo, chatID, 6102)

	// 6101: a worker took it and died; its lock has expired.
	_, err := pool.Exec(
		ctx,
		`
		UPDATE photo_jobs
		SET status = 'processing', attempts = 1,
		    locked_until = NOW() - INTERVAL '1 minute'
		WHERE chat_id = $1 AND telegram_message_id = 6101
		`,
		chatID,
	)
	if err != nil {
		t.Fatalf("make job abandoned: %v", err)
	}

	// 6102: waiting for a retry later.
	_, err = pool.Exec(
		ctx,
		`
		UPDATE photo_jobs
		SET next_attempt_at = NOW() + INTERVAL '1 hour'
		WHERE chat_id = $1 AND telegram_message_id = 6102
		`,
		chatID,
	)
	if err != nil {
		t.Fatalf("delay job: %v", err)
	}

	job, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim abandoned job: %v", err)
	}
	if job == nil || job.TelegramMessageID != 6101 {
		t.Fatalf("claim = %v, want abandoned job 6101", job)
	}
	if job.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", job.Attempts)
	}

	next, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim delayed job: %v", err)
	}
	if next != nil {
		t.Fatalf("claimed delayed job %d before its time, want nil", next.TelegramMessageID)
	}
}

func TestClaimPhotoJobConcurrentWorkers(t *testing.T) {
	chatID := -randomTestTgID()
	repo, pool := newPhotoJobTestRepo(t, chatID)
	ctx := context.Background()

	requireNoOtherReadyJobs(t, pool, chatID)

	const jobs = 5
	const workers = 8

	for i := 0; i < jobs; i++ {
		enqueueTestJob(t, repo, chatID, 6200+i)
	}

	var (
		mu      sync.Mutex
		claimed = map[int64]int{}
		wg      sync.WaitGroup
	)

	start := make(chan struct{})

	for w := 0; w < workers; w++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for {
				job, err := repo.ClaimPhotoJob(ctx, time.Minute)
				if err != nil {
					t.Errorf("claim: %v", err)
					return
				}
				if job == nil {
					return
				}

				mu.Lock()
				claimed[job.ID]++
				mu.Unlock()
			}
		}()
	}

	close(start)
	wg.Wait()

	if len(claimed) != jobs {
		t.Fatalf("claimed %d different jobs, want %d", len(claimed), jobs)
	}

	for id, times := range claimed {
		if times != 1 {
			t.Fatalf("job %d was claimed %d times, want 1", id, times)
		}
	}
}

// requireNoOtherReadyJobs stops the test if jobs from other chats could be
// claimed, because ClaimPhotoJob takes any ready job in the table.
func requireNoOtherReadyJobs(t *testing.T, pool *pgxpool.Pool, chatID int64) {
	t.Helper()

	var other int
	err := pool.QueryRow(
		context.Background(),
		`
		SELECT COUNT(*)
		FROM photo_jobs
		WHERE chat_id <> $1
		  AND status IN ('pending', 'processing')
		`,
		chatID,
	).Scan(&other)
	if err != nil {
		t.Fatalf("count other jobs: %v", err)
	}
	if other != 0 {
		t.Fatalf("test database has %d unfinished jobs from other chats; clean photo_jobs first", other)
	}
}

func TestMarkPhotoJobDone(t *testing.T) {
	chatID := -randomTestTgID()
	repo, pool := newPhotoJobTestRepo(t, chatID)
	ctx := context.Background()

	requireNoOtherReadyJobs(t, pool, chatID)

	enqueueTestJob(t, repo, chatID, 6301)

	job, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil || job == nil {
		t.Fatalf("claim = (%v, %v), want a job", job, err)
	}

	if err := repo.MarkPhotoJobDone(ctx, job.ID); err != nil {
		t.Fatalf("mark done: %v", err)
	}

	var (
		status      string
		lockedUntil *time.Time
		updatedAt   time.Time
	)

	err = pool.QueryRow(
		ctx,
		`SELECT status, locked_until, updated_at FROM photo_jobs WHERE id = $1`,
		job.ID,
	).Scan(&status, &lockedUntil, &updatedAt)
	if err != nil {
		t.Fatalf("read job: %v", err)
	}

	if status != models.PhotoJobDone {
		t.Fatalf("status = %q, want %q", status, models.PhotoJobDone)
	}
	if lockedUntil != nil {
		t.Fatalf("locked_until = %v, want NULL", *lockedUntil)
	}
	if updatedAt.Before(job.UpdatedAt) {
		t.Fatalf("updated_at = %v, want not before %v", updatedAt, job.UpdatedAt)
	}

	// A finished job is never handed out again.
	next, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim after done: %v", err)
	}
	if next != nil {
		t.Fatalf("claimed finished job %d, want nil", next.ID)
	}

	// An unknown job is reported as pgx.ErrNoRows.
	err = repo.MarkPhotoJobDone(ctx, -1)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("mark unknown job: error = %v, want pgx.ErrNoRows", err)
	}
}

func TestMarkPhotoJobDead(t *testing.T) {
	chatID := -randomTestTgID()
	repo, pool := newPhotoJobTestRepo(t, chatID)
	ctx := context.Background()

	requireNoOtherReadyJobs(t, pool, chatID)

	enqueueTestJob(t, repo, chatID, 6401)

	job, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil || job == nil {
		t.Fatalf("claim = (%v, %v), want a job", job, err)
	}

	const reason = "download failed: context deadline exceeded"

	if err := repo.MarkPhotoJobDead(ctx, job.ID, reason); err != nil {
		t.Fatalf("mark dead: %v", err)
	}

	var (
		status      string
		lockedUntil *time.Time
		lastError   string
		attempts    int
	)

	err = pool.QueryRow(
		ctx,
		`SELECT status, locked_until, last_error, attempts FROM photo_jobs WHERE id = $1`,
		job.ID,
	).Scan(&status, &lockedUntil, &lastError, &attempts)
	if err != nil {
		t.Fatalf("read job: %v", err)
	}

	if status != models.PhotoJobDead {
		t.Fatalf("status = %q, want %q", status, models.PhotoJobDead)
	}
	if lockedUntil != nil {
		t.Fatalf("locked_until = %v, want NULL", *lockedUntil)
	}
	if lastError != reason {
		t.Fatalf("last_error = %q, want %q", lastError, reason)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1 (marking dead must not change it)", attempts)
	}

	// A dead job is never handed out again.
	next, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim after dead: %v", err)
	}
	if next != nil {
		t.Fatalf("claimed dead job %d, want nil", next.ID)
	}

	err = repo.MarkPhotoJobDead(ctx, -1, reason)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("mark unknown job dead: error = %v, want pgx.ErrNoRows", err)
	}
}

func TestRetryPhotoJob(t *testing.T) {
	chatID := -randomTestTgID()
	repo, pool := newPhotoJobTestRepo(t, chatID)
	ctx := context.Background()

	requireNoOtherReadyJobs(t, pool, chatID)

	enqueueTestJob(t, repo, chatID, 6501)

	job, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil || job == nil {
		t.Fatalf("claim = (%v, %v), want a job", job, err)
	}

	const reason = "telegram: timeout"

	before := time.Now()

	if err := repo.RetryPhotoJob(ctx, job.ID, 30*time.Second, reason); err != nil {
		t.Fatalf("retry: %v", err)
	}

	var (
		status        string
		lockedUntil   *time.Time
		lastError     string
		attempts      int
		nextAttemptAt time.Time
	)

	err = pool.QueryRow(
		ctx,
		`
		SELECT status, locked_until, last_error, attempts, next_attempt_at
		FROM photo_jobs
		WHERE id = $1
		`,
		job.ID,
	).Scan(&status, &lockedUntil, &lastError, &attempts, &nextAttemptAt)
	if err != nil {
		t.Fatalf("read job: %v", err)
	}

	if status != models.PhotoJobPending {
		t.Fatalf("status = %q, want %q", status, models.PhotoJobPending)
	}
	if lockedUntil != nil {
		t.Fatalf("locked_until = %v, want NULL", *lockedUntil)
	}
	if lastError != reason {
		t.Fatalf("last_error = %q, want %q", lastError, reason)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1 (only Claim counts attempts)", attempts)
	}

	// The next attempt must be about 30 seconds ahead, not seconds and not years.
	delay := nextAttemptAt.Sub(before)
	if delay < 20*time.Second || delay > 40*time.Second {
		t.Fatalf("next attempt in %v, want about 30s", delay)
	}

	// Before its time the job is not handed out.
	next, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim before retry time: %v", err)
	}
	if next != nil {
		t.Fatalf("claimed job %d before its retry time, want nil", next.ID)
	}

	// Once the time has come, the job is claimed again as a second attempt.
	_, err = pool.Exec(
		ctx,
		`UPDATE photo_jobs SET next_attempt_at = NOW() - INTERVAL '1 second' WHERE id = $1`,
		job.ID,
	)
	if err != nil {
		t.Fatalf("move retry time to the past: %v", err)
	}

	again, err := repo.ClaimPhotoJob(ctx, time.Minute)
	if err != nil {
		t.Fatalf("claim after retry time: %v", err)
	}
	if again == nil || again.ID != job.ID {
		t.Fatalf("claim = %v, want job %d", again, job.ID)
	}
	if again.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", again.Attempts)
	}

	err = repo.RetryPhotoJob(ctx, -1, time.Second, reason)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("retry unknown job: error = %v, want pgx.ErrNoRows", err)
	}
}
