package service

import (
	"context"
	"errors"
	"testing"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5"
)

type userServiceRepoMock struct {
	user         *models.User
	userByID     *models.User
	getErr       error
	getByIDErr   error
	createErr    error
	updateErr    error
	assignErr    error
	updateCalled bool
	assignCalled bool

	updatedTgID   int64
	updatedStatus string
	updatedRole   *string

	assignedUserID   string
	assignedSRCodeID int

	workGroups     []models.WorkGroup
	territories    []models.Territory
	srCodes        []models.SRCode
	workGroupsErr  error
	territoriesErr error
	srCodesErr     error

	assignAndApproveErr    error
	assignAndApproveCalled bool
	assignedTgID           int64

	assignSVErr error
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

func (m *userServiceRepoMock) GetUserByID(
	_ context.Context,
	_ string,
) (*models.User, error) {
	return m.userByID, m.getByIDErr
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

func (m *userServiceRepoMock) AssignSRCode(
	_ context.Context,
	userID string,
	srCodeID int,
) error {
	m.assignCalled = true
	m.assignedUserID = userID
	m.assignedSRCodeID = srCodeID
	return m.assignErr
}

func (m *userServiceRepoMock) GetAssignmentsByUser(
	_ context.Context,
	_ string,
) ([]*models.SRAssignment, error) {
	return nil, nil
}

func (m *userServiceRepoMock) GetWorkGroups(
	_ context.Context,
) ([]models.WorkGroup, error) {
	return m.workGroups, m.workGroupsErr
}

func (m *userServiceRepoMock) GetTerritoriesByGroup(
	_ context.Context,
	_ int,
) ([]models.Territory, error) {
	return m.territories, m.territoriesErr
}

func (m *userServiceRepoMock) GetAvailableSRCodes(
	_ context.Context,
	_ int,
) ([]models.SRCode, error) {
	return m.srCodes, m.srCodesErr
}

func TestGetUserByTgIDSuccess(t *testing.T) {
	expected := &models.User{
		TgID:     8281761514,
		FullName: "Muhammad",
	}

	service := NewUserService(&userServiceRepoMock{user: expected})

	user, err := service.GetUserByTgID(
		context.Background(),
		expected.TgID,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user != expected {
		t.Fatalf("user = %+v, want %+v", user, expected)
	}
}

func TestGetUserByTgIDNotFound(t *testing.T) {
	repo := &userServiceRepoMock{getErr: pgx.ErrNoRows}
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

func TestCreateUserRepositoryError(t *testing.T) {
	expectedErr := errors.New("database error")
	repo := &userServiceRepoMock{createErr: expectedErr}
	service := NewUserService(repo)

	err := service.CreateUser(context.Background(), &models.User{})

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

	if repo.updatedStatus != "approved" {
		t.Errorf("status = %q, want %q",
			repo.updatedStatus, "approved")
	}

	if repo.updatedRole == nil ||
		*repo.updatedRole != "Dispatcher" {
		t.Errorf("role = %v, want Dispatcher",
			repo.updatedRole)
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
		t.Errorf("status = %q, want rejected",
			repo.updatedStatus)
	}

	if repo.updatedRole != nil {
		t.Fatal("role should be nil")
	}
}

func TestGetUserByIDSuccess(t *testing.T) {
	expected := &models.User{
		ID:       "user-1",
		FullName: "Muhammad",
	}

	repo := &userServiceRepoMock{userByID: expected}
	service := NewUserService(repo)

	user, err := service.GetUserByID(
		context.Background(),
		"user-1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user != expected {
		t.Fatalf("user = %+v, want %+v", user, expected)
	}
}

func TestAssignSRCodeSuccess(t *testing.T) {
	role := "Торговый представитель"

	repo := &userServiceRepoMock{
		userByID: &models.User{
			ID:     "user-1",
			Role:   &role,
			Status: "approved",
		},
	}

	service := NewUserService(repo)

	err := service.AssignSRCode(
		context.Background(),
		"user-1",
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.assignCalled {
		t.Fatal("AssignSRCodeAndUpdate was not called")
	}
}

func TestAssignSRCodeNotApproved(t *testing.T) {
	role := "Торговый представитель"

	repo := &userServiceRepoMock{
		userByID: &models.User{
			Role:   &role,
			Status: "pending",
		},
	}

	service := NewUserService(repo)

	err := service.AssignSRCode(
		context.Background(),
		"user-1",
		10,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if repo.assignCalled {
		t.Fatal("AssignSRCodeAndUpdate should not be called")
	}
}

func TestAssignSRCodeWrongRole(t *testing.T) {
	role := "Супервайзер"

	repo := &userServiceRepoMock{
		userByID: &models.User{
			Role:   &role,
			Status: "approved",
		},
	}

	service := NewUserService(repo)

	err := service.AssignSRCode(
		context.Background(),
		"user-1",
		10,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if repo.assignCalled {
		t.Fatal("AssignSRCodeAndUpdate should not be called")
	}
}

func TestGetWorkGroups(t *testing.T) {
	expected := []models.WorkGroup{
		{ID: 1, Name: "Розница"},
		{ID: 2, Name: "ОПТ"},
	}

	repo := &userServiceRepoMock{workGroups: expected}
	service := NewUserService(repo)

	result, err := service.GetWorkGroups(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("got %d groups, want 2", len(result))
	}
}

func TestGetTerritoriesByGroup(t *testing.T) {
	repo := &userServiceRepoMock{
		territories: []models.Territory{
			{ID: 1, Name: "Розница 1", WorkGroupID: 1},
		},
	}

	service := NewUserService(repo)

	result, err := service.GetTerritoriesByGroup(
		context.Background(),
		1,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("got %d territories, want 1", len(result))
	}
}

func TestGetAvailableSRCodes(t *testing.T) {
	repo := &userServiceRepoMock{
		srCodes: []models.SRCode{
			{ID: 1, Code: "SAM-1"},
		},
	}

	service := NewUserService(repo)

	result, err := service.GetAvailableSRCodes(
		context.Background(),
		1,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("got %d codes, want 1", len(result))
	}

	if result[0].Code != "SAM-1" {
		t.Errorf("code = %q, want SAM-1", result[0].Code)
	}
}

func (m *userServiceRepoMock) AssignSRCodeAndUpdate(
	_ context.Context,
	tgID int64,
	_ string,
	_ *string,
	userID string,
	srCodeID int,
) error {
	m.assignAndApproveCalled = true
	m.assignedTgID = tgID
	m.assignedUserID = userID
	m.assignedSRCodeID = srCodeID

	return m.assignAndApproveErr
}

func TestAssignSRCodeAndApproveRepositoryError(t *testing.T) {
	expectedErr := errors.New("database error")

	repo := &userServiceRepoMock{
		userByID: &models.User{
			ID: "user-1",
		},
		assignAndApproveErr: expectedErr,
	}

	service := NewUserService(repo)

	err := service.AssignSRCodeAndApprove(
		context.Background(),
		"user-1",
		10,
		8281761514,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestAssignSRCodeAndApproveSuccess(t *testing.T) {
	repo := &userServiceRepoMock{
		userByID: &models.User{
			ID: "user-1",
		},
	}

	service := NewUserService(repo)

	err := service.AssignSRCodeAndApprove(
		context.Background(),
		"user-1",
		10,
		8281761514,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.assignAndApproveCalled {
		t.Fatal("AssignSRCodeAndUpdate was not called")
	}
}

func (m *userServiceRepoMock) AssignSVTerritoryAndUpdate(
	_ context.Context,
	_ int64,
	_ string,
	_ *string,
	_ string,
	_ int,
) error {
	return m.assignSVErr
}
