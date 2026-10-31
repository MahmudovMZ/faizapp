package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/repository/postgres"
	"github.com/jackc/pgx/v5"
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

	result, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
		"Test User",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 5 {
		t.Fatalf("count = %d, want 5", result.Count)
	}

	if result.NewSender {
		t.Fatal("NewSender = true, want false for an existing user")
	}

	if repo.pendingCalled {
		t.Fatal("CreatePendingUser must not be called for an existing user")
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
		"Test User",
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
		"Test User",
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
		"Test User",
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
		"Test User",
	)

	if !errors.Is(err, ErrDuplicatePhoto) {
		t.Fatalf(
			"error = %v, want ErrDuplicatePhoto",
			err,
		)
	}
}

func TestSavePhotoUnknownSenderCreatesPendingUser(t *testing.T) {
	repo := &userServiceRepoMock{
		getErr: pgx.ErrNoRows,
		pendingUser: &models.User{
			ID:     "user-new",
			Status: "pending",
		},
		photoCount: 1,
	}

	service := NewPhotoService(repo)

	result, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
		"Test User",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.pendingCalled {
		t.Fatal("expected CreatePendingUser to be called")
	}

	if repo.pendingTgID != 123456789 {
		t.Fatalf("pending tg_id = %d, want 123456789", repo.pendingTgID)
	}

	if repo.pendingFullName != "Test User" {
		t.Fatalf("pending full name = %q, want %q", repo.pendingFullName, "Test User")
	}

	if result.Count != 1 {
		t.Fatalf("count = %d, want 1", result.Count)
	}

	if !result.NewSender {
		t.Fatal("NewSender = false, want true for an unknown sender")
	}

	if result.Counted {
		t.Fatal("Counted = true, want false for a user without a role")
	}
}

func TestSavePhotoUnknownSenderCreateError(t *testing.T) {
	expectedErr := errors.New("create pending user error")

	repo := &userServiceRepoMock{
		getErr:     pgx.ErrNoRows,
		pendingErr: expectedErr,
	}

	service := NewPhotoService(repo)

	_, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
		"Test User",
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestSavePhotoGetUserError(t *testing.T) {
	expectedErr := errors.New("database is down")

	repo := &userServiceRepoMock{
		getErr: expectedErr,
	}

	service := NewPhotoService(repo)

	_, err := service.SavePhoto(
		context.Background(),
		123456789,
		10,
		[]byte("photo data"),
		"",
		time.Now(),
		"Test User",
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}

	if repo.pendingCalled {
		t.Fatal("CreatePendingUser must not be called on other errors")
	}
}

func TestSavePhotoCountedByRole(t *testing.T) {
	salesRep := models.RoleSalesRep
	supervisor := models.RoleSupervisor
	dispatcher := models.RoleDispatcher
	seniorSupervisor := models.RoleSeniorSupervisor

	tests := []struct {
		name string
		role *string
		want bool
	}{
		{name: "sales representative", role: &salesRep, want: true},
		{name: "supervisor", role: &supervisor, want: true},
		{name: "dispatcher", role: &dispatcher, want: false},
		{name: "senior supervisor", role: &seniorSupervisor, want: false},
		{name: "no role yet", role: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userServiceRepoMock{
				user: &models.User{
					ID:   "user-1",
					Role: tt.role,
				},
				photoCount: 3,
			}

			service := NewPhotoService(repo)

			result, err := service.SavePhoto(
				context.Background(),
				123456789,
				10,
				[]byte("photo data"),
				"",
				time.Now(),
				"Test User",
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Counted != tt.want {
				t.Fatalf("Counted = %v, want %v", result.Counted, tt.want)
			}
		})
	}
}
