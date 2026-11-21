package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
)

type PhotoJobStore interface {
	EnqueuePhotoJob(ctx context.Context, job *models.PhotoJob) error
	ClaimPhotoJob(ctx context.Context, lockFor time.Duration) (*models.PhotoJob, error)
	MarkPhotoJobDone(ctx context.Context, id int64) error
	MarkPhotoJobDead(ctx context.Context, id int64, errTxt string) error
	RetryPhotoJob(ctx context.Context, id int64, retryAfter time.Duration, errTxt string) error
}

type PhotoJobService struct {
	store   PhotoJobStore
	lockFor time.Duration
}

const (
	maxAttempts = 5
	retryDelay  = 15 * time.Second
)

func NewPhotoJobService(store PhotoJobStore, lockFor time.Duration) *PhotoJobService {
	return &PhotoJobService{store: store, lockFor: lockFor}
}

func (s *PhotoJobService) EnqueueJob(ctx context.Context, job *models.PhotoJob) error {
	err := s.store.EnqueuePhotoJob(ctx, job)
	if err != nil {
		return fmt.Errorf("enqueue photo job: %w", err)
	}
	return nil
}

func (s *PhotoJobService) NextJob(ctx context.Context) (*models.PhotoJob, error) {
	job, err := s.store.ClaimPhotoJob(ctx, s.lockFor)
	if err != nil {
		return nil, fmt.Errorf("claim photo job: %w", err)
	}
	return job, nil
}

func (s *PhotoJobService) FinishJob(ctx context.Context, job *models.PhotoJob, processErr error) error {
	if processErr == nil || errors.Is(processErr, ErrDuplicatePhoto) {
		err := s.store.MarkPhotoJobDone(ctx, job.ID)
		if err != nil {
			return fmt.Errorf("mark photo job done: %w", err)
		}
		return nil
	}

	if job.Attempts >= maxAttempts {
		if err := s.store.MarkPhotoJobDead(ctx, job.ID, processErr.Error()); err != nil {
			return fmt.Errorf("mark photo job dead: %w", err)
		}
		return nil
	}

	err := s.store.RetryPhotoJob(ctx, job.ID, retryDelay, processErr.Error())
	if err != nil {
		return fmt.Errorf("retry photo job: %w", err)
	}
	return nil
}
