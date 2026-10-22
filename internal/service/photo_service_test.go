package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/repository/postgres"
)

func TestCalculateSHA256(t *testing.T) {
	data := []byte("hello")

	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	result := CalcSHA256(data)

	if result != expected {
		t.Fatalf(
			"hash = %s, want %s",
			result,
			expected,
		)
	}
}

func (m *userServiceRepoMock) CreatePhoto(
	_ context.Context,
	_ *models.Photo,
) error {
	return m.photoErr
}

func (m *userServiceRepoMock) CountPhotosByUser(
	_ context.Context,
	_ string,
) (int, error) {
	return m.photoCount, m.photoCountErr
}

func (m *userServiceRepoMock) CountPhotosByDate(
	_ context.Context,
	_ time.Time,
	_ time.Time,
) (int, error) {
	return m.photoCount, m.photoCountErr
}

func TestSavePhotoSuccess(t *testing.T) {
	repo := &userServiceRepoMock{
		user: &models.User{
			ID: "user-1",
		},
		photoCount: 5,
	}

	service := NewPhotoService(repo)

	count, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 5 {
		t.Fatalf("count = %d, want 5", count)
	}
}

func TestSavePhotoUserNotFound(t *testing.T) {
	repo := &userServiceRepoMock{}

	service := NewPhotoService(repo)

	_, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSavePhotoCreateError(t *testing.T) {
	expectedErr := errors.New("create photo error")

	repo := &userServiceRepoMock{
		user: &models.User{
			ID: "user-1",
		},
		photoErr: expectedErr,
	}

	service := NewPhotoService(repo)

	_, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestSavePhotoCountError(t *testing.T) {
	expectedErr := errors.New("count photos error")

	repo := &userServiceRepoMock{
		user: &models.User{
			ID: "user-1",
		},
		photoCountErr: expectedErr,
	}

	service := NewPhotoService(repo)

	_, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}
func TestSavePhotoDuplicate(t *testing.T) {
	repo := &userServiceRepoMock{
		user: &models.User{
			ID: "user-1",
		},
		photoErr: postgres.ErrDuplicatePhoto,
	}

	service := NewPhotoService(repo)

	_, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
	)

	if !errors.Is(err, ErrDuplicatePhoto) {
		t.Fatalf(
			"error = %v, want ErrDuplicatePhoto",
			err,
		)
	}
}
