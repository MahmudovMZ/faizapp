package service

import (
	"context"
	"errors"
	"testing"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5"
)

type mockUserRepository struct {
	user         *models.User
	getErr       error
	createErr    error
	getCalled    bool
	createCalled bool
}

func stringPtr(value string) *string {
	return &value
}

func (m *mockUserRepository) GetUserByTgID(
	ctx context.Context,
	tgID int64,
) (*models.User, error) {
	m.getCalled = true
	return m.user, m.getErr
}

func (m *mockUserRepository) CreateUser(
	ctx context.Context,
	user *models.User,
) error {
	m.createCalled = true
	return m.createErr
}

func TestUserService_GetUserByTgID_UserFound(t *testing.T) {
	expectedUser := &models.User{
		ID:       "user-id",
		TgID:     12345,
		FullName: "Test User",
		Role:     stringPtr("dispatcher"),
	}

	repo := &mockUserRepository{
		user: expectedUser,
	}

	service := NewUserService(repo)

	result, err := service.GetUserByTgID(context.Background(), 12345)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != expectedUser {
		t.Fatalf("expected user %v, got %v", expectedUser, result)
	}

	if !repo.getCalled {
		t.Fatal("expected repository GetUserByTgID to be called")
	}
}

func TestUserService_GetUserByTgID_UserNotFound(t *testing.T) {
	repo := &mockUserRepository{
		getErr: pgx.ErrNoRows,
	}

	service := NewUserService(repo)

	result, err := service.GetUserByTgID(context.Background(), 12345)
	if result != nil {
		t.Fatalf("expected nil user, got %v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestUserService_GetUserByTgID_RepositoryError(t *testing.T) {
	expectedErr := errors.New("database error")

	repo := &mockUserRepository{
		getErr: expectedErr,
	}

	service := NewUserService(repo)

	result, err := service.GetUserByTgID(context.Background(), 12345)
	if result != nil {
		t.Fatalf("expected nil user, got %v", result)
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestUserService_CreateUser_Success(t *testing.T) {
	repo := &mockUserRepository{}

	service := NewUserService(repo)

	user := &models.User{
		TgID:     12345,
		FullName: "Test User",
		Role:     stringPtr("dispatcher"),
	}

	err := service.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.createCalled {
		t.Fatal("expected repository CreateUser to be called")
	}
}

func TestUserService_CreateUser_Error(t *testing.T) {
	expectedErr := errors.New("database error")

	repo := &mockUserRepository{
		createErr: expectedErr,
	}

	service := NewUserService(repo)

	user := &models.User{
		TgID:     12345,
		FullName: "Test User",
		Role:     stringPtr("dispatcher"),
	}

	err := service.CreateUser(context.Background(), user)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
