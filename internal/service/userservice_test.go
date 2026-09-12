package service

import (
	"context"
	"errors"
	"testing"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5"
)

type userServiceRepoMock struct {
	user          *models.User
	getErr        error
	createErr     error
	updateErr     error
	updateCalled  bool
	updatedTgID   int64
	updatedStatus string
	updatedRole   *string
}

func (m *userServiceRepoMock) CreateUser(
	_ context.Context,
	_ *models.User,
) error {
	return m.createErr
}

func (m *userServiceRepoMock) GetUserByTgID(
	_ context.Context,
	_ int64,
) (*models.User, error) {
	return m.user, m.getErr
}

func (m *userServiceRepoMock) UpdateUserStatus(
	_ context.Context,
	tgID int64,
	status string,
	role *string,
) error {
	m.updateCalled = true
	m.updatedTgID = tgID
	m.updatedStatus = status
	m.updatedRole = role

	return m.updateErr
}

func TestGetUserByTgIDSuccess(t *testing.T) {
	expectedUser := &models.User{
		TgID:     8281761514,
		FullName: "Muhammad",
	}

	repo := &userServiceRepoMock{
		user: expectedUser,
	}

	service := NewUserService(repo)

	user, err := service.GetUserByTgID(
		context.Background(),
		expectedUser.TgID,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user != expectedUser {
		t.Fatalf("user = %+v, want %+v", user, expectedUser)
	}
}

func TestGetUserByTgIDNotFound(t *testing.T) {
	repo := &userServiceRepoMock{
		getErr: pgx.ErrNoRows,
	}

	service := NewUserService(repo)

	user, err := service.GetUserByTgID(
		context.Background(),
		8281761514,
	)

	if user != nil {
		t.Fatalf("user = %+v, want nil", user)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("error = %v, want pgx.ErrNoRows", err)
	}
}

func TestGetUserByTgIDRepositoryError(t *testing.T) {
	expectedErr := errors.New("database error")

	repo := &userServiceRepoMock{
		getErr: expectedErr,
	}

	service := NewUserService(repo)

	_, err := service.GetUserByTgID(
		context.Background(),
		8281761514,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestCreateUserSuccess(t *testing.T) {
	repo := &userServiceRepoMock{}
	service := NewUserService(repo)

	user := &models.User{
		TgID:     8281761514,
		FullName: "Muhammad",
	}

	err := service.CreateUser(
		context.Background(),
		user,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateUserRepositoryError(t *testing.T) {
	expectedErr := errors.New("database error")

	repo := &userServiceRepoMock{
		createErr: expectedErr,
	}

	service := NewUserService(repo)

	err := service.CreateUser(
		context.Background(),
		&models.User{},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestApproveUserSuccess(t *testing.T) {
	repo := &userServiceRepoMock{}
	service := NewUserService(repo)

	err := service.ApproveUser(
		context.Background(),
		8281761514,
		"Dispatcher",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.updateCalled {
		t.Fatal("UpdateUserStatus was not called")
	}

	if repo.updatedTgID != 8281761514 {
		t.Errorf(
			"tgID = %d, want %d",
			repo.updatedTgID,
			8281761514,
		)
	}

	if repo.updatedStatus != "approved" {
		t.Errorf(
			"status = %q, want %q",
			repo.updatedStatus,
			"approved",
		)
	}

	if repo.updatedRole == nil {
		t.Fatal("role is nil")
	}

	if *repo.updatedRole != "Dispatcher" {
		t.Errorf(
			"role = %q, want %q",
			*repo.updatedRole,
			"Dispatcher",
		)
	}
}

func TestApproveUserEmptyRole(t *testing.T) {
	repo := &userServiceRepoMock{}
	service := NewUserService(repo)

	err := service.ApproveUser(
		context.Background(),
		8281761514,
		"",
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if repo.updateCalled {
		t.Fatal("UpdateUserStatus should not be called")
	}
}

func TestApproveUserRepositoryError(t *testing.T) {
	expectedErr := errors.New("database error")

	repo := &userServiceRepoMock{
		updateErr: expectedErr,
	}

	service := NewUserService(repo)

	err := service.ApproveUser(
		context.Background(),
		8281761514,
		"Dispatcher",
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestRejectUserSuccess(t *testing.T) {
	repo := &userServiceRepoMock{}
	service := NewUserService(repo)

	err := service.RejectUser(
		context.Background(),
		8281761514,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.updateCalled {
		t.Fatal("UpdateUserStatus was not called")
	}

	if repo.updatedStatus != "rejected" {
		t.Errorf(
			"status = %q, want %q",
			repo.updatedStatus,
			"rejected",
		)
	}

	if repo.updatedRole != nil {
		t.Fatal("role should be nil")
	}
}

func TestRejectUserRepositoryError(t *testing.T) {
	expectedErr := errors.New("database error")

	repo := &userServiceRepoMock{
		updateErr: expectedErr,
	}

	service := NewUserService(repo)

	err := service.RejectUser(
		context.Background(),
		8281761514,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}
