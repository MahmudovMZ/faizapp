package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
)

// photoJobStoreMock implements only the five queue methods the service needs.
type photoJobStoreMock struct {
	enqueueErr  error
	enqueuedJob *models.PhotoJob

	claimJob     *models.PhotoJob
	claimErr     error
	claimLockFor time.Duration

	doneErr error
	doneID  int64

	deadErr  error
	deadID   int64
	deadText string

	retryErr   error
	retryID    int64
	retryAfter time.Duration
	retryText  string

	// calls records which store method was used, in order.
	calls []string
}

func (m *photoJobStoreMock) EnqueuePhotoJob(_ context.Context, job *models.PhotoJob) error {
	m.calls = append(m.calls, "enqueue")
	m.enqueuedJob = job

	return m.enqueueErr
}

func (m *photoJobStoreMock) ClaimPhotoJob(_ context.Context, lockFor time.Duration) (*models.PhotoJob, error) {
	m.calls = append(m.calls, "claim")
	m.claimLockFor = lockFor

	return m.claimJob, m.claimErr
}

func (m *photoJobStoreMock) MarkPhotoJobDone(_ context.Context, id int64) error {
	m.calls = append(m.calls, "done")
	m.doneID = id

	return m.doneErr
}

func (m *photoJobStoreMock) MarkPhotoJobDead(_ context.Context, id int64, errText string) error {
	m.calls = append(m.calls, "dead")
	m.deadID = id
	m.deadText = errText

	return m.deadErr
}

func (m *photoJobStoreMock) RetryPhotoJob(_ context.Context, id int64, retryAfter time.Duration, errText string) error {
	m.calls = append(m.calls, "retry")
	m.retryID = id
	m.retryAfter = retryAfter
	m.retryText = errText

	return m.retryErr
}

func (m *photoJobStoreMock) onlyCalled(t *testing.T, want string) {
	t.Helper()

	if len(m.calls) != 1 || m.calls[0] != want {
		t.Fatalf("store calls = %v, want only %q", m.calls, want)
	}
}

func TestPhotoJobServiceEnqueueJob(t *testing.T) {
	store := &photoJobStoreMock{}
	service := NewPhotoJobService(store, time.Minute)

	job := &models.PhotoJob{ChatID: 1, TelegramMessageID: 2, FileID: "file"}

	if err := service.EnqueueJob(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store.onlyCalled(t, "enqueue")

	if store.enqueuedJob != job {
		t.Fatal("the job was not passed to the store")
	}
}

func TestPhotoJobServiceEnqueueJobError(t *testing.T) {
	dbErr := errors.New("database is down")
	store := &photoJobStoreMock{enqueueErr: dbErr}
	service := NewPhotoJobService(store, time.Minute)

	err := service.EnqueueJob(context.Background(), &models.PhotoJob{})
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

func TestPhotoJobServiceNextJob(t *testing.T) {
	ready := &models.PhotoJob{ID: 7}
	store := &photoJobStoreMock{claimJob: ready}
	service := NewPhotoJobService(store, 90*time.Second)

	job, err := service.NextJob(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if job != ready {
		t.Fatalf("job = %v, want job 7", job)
	}

	if store.claimLockFor != 90*time.Second {
		t.Fatalf("lock = %v, want the service lock 1m30s", store.claimLockFor)
	}
}

func TestPhotoJobServiceNextJobEmptyQueue(t *testing.T) {
	store := &photoJobStoreMock{}
	service := NewPhotoJobService(store, time.Minute)

	job, err := service.NextJob(context.Background())
	if err != nil {
		t.Fatalf("empty queue: unexpected error: %v", err)
	}
	if job != nil {
		t.Fatalf("empty queue: job = %v, want nil", job)
	}
}

func TestPhotoJobServiceNextJobError(t *testing.T) {
	dbErr := errors.New("database is down")
	store := &photoJobStoreMock{claimErr: dbErr}
	service := NewPhotoJobService(store, time.Minute)

	job, err := service.NextJob(context.Background())
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
	if job != nil {
		t.Fatalf("job = %v, want nil on error", job)
	}
}

func TestPhotoJobServiceFinishJob(t *testing.T) {
	downloadErr := errors.New("download failed: context deadline exceeded")
	duplicateErr := fmt.Errorf("[HANDLER] save photo failed: %w", ErrDuplicatePhoto)

	tests := []struct {
		name       string
		attempts   int
		processErr error
		wantCall   string
	}{
		{name: "success", attempts: 1, processErr: nil, wantCall: "done"},
		{name: "duplicate photo is final", attempts: 1, processErr: duplicateErr, wantCall: "done"},
		{name: "first failure is retried", attempts: 1, processErr: downloadErr, wantCall: "retry"},
		{name: "failure before the limit is retried", attempts: maxAttempts - 1, processErr: downloadErr, wantCall: "retry"},
		{name: "failure at the limit is dead", attempts: maxAttempts, processErr: downloadErr, wantCall: "dead"},
		{name: "failure above the limit is dead", attempts: maxAttempts + 1, processErr: downloadErr, wantCall: "dead"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &photoJobStoreMock{}
			service := NewPhotoJobService(store, time.Minute)

			job := &models.PhotoJob{ID: 42, Attempts: tt.attempts}

			if err := service.FinishJob(context.Background(), job, tt.processErr); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			store.onlyCalled(t, tt.wantCall)

			switch tt.wantCall {
			case "done":
				if store.doneID != 42 {
					t.Fatalf("done id = %d, want 42", store.doneID)
				}
			case "dead":
				if store.deadID != 42 || store.deadText != downloadErr.Error() {
					t.Fatalf("dead = (%d, %q), want (42, %q)", store.deadID, store.deadText, downloadErr.Error())
				}
			case "retry":
				if store.retryID != 42 || store.retryText != downloadErr.Error() {
					t.Fatalf("retry = (%d, %q), want (42, %q)", store.retryID, store.retryText, downloadErr.Error())
				}
				if store.retryAfter != retryDelay {
					t.Fatalf("retry after = %v, want %v", store.retryAfter, retryDelay)
				}
			}
		})
	}
}

func TestPhotoJobServiceFinishJobStoreErrors(t *testing.T) {
	dbErr := errors.New("database is down")
	processErr := errors.New("download failed")

	tests := []struct {
		name       string
		store      *photoJobStoreMock
		attempts   int
		processErr error
	}{
		{name: "done fails", store: &photoJobStoreMock{doneErr: dbErr}, attempts: 1, processErr: nil},
		{name: "retry fails", store: &photoJobStoreMock{retryErr: dbErr}, attempts: 1, processErr: processErr},
		{name: "dead fails", store: &photoJobStoreMock{deadErr: dbErr}, attempts: maxAttempts, processErr: processErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPhotoJobService(tt.store, time.Minute)
			job := &models.PhotoJob{ID: 1, Attempts: tt.attempts}

			err := service.FinishJob(context.Background(), job, tt.processErr)
			if !errors.Is(err, dbErr) {
				t.Fatalf("error = %v, want %v", err, dbErr)
			}
		})
	}
}
