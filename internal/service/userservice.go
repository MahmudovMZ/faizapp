package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/repository/postgres"
	"github.com/jackc/pgx/v5"
)

type UserService struct {
	repo postgres.Repository
}

func NewUserService(repo postgres.Repository) *UserService {
	return &UserService{repo: repo}
}
func (u *UserService) GetUserByTgID(ctx context.Context, tgID int64) (*models.User, error) {
	user, err := u.repo.GetUserByTgID(ctx, tgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, err
	}
	return user, nil
}

func (u *UserService) CreateUser(ctx context.Context, user *models.User) error {
	err := u.repo.CreateUser(ctx, user)
	if err != nil {
		return err
	}
	return nil
}
func (u *UserService) ApproveUser(ctx context.Context, tgID int64, role string) error {
	role = strings.TrimSpace(role)
	if role == "" {
		return errors.New("role cannot be empty")
	}
	return u.repo.UpdateUserStatus(
		ctx,
		tgID,
		"approved",
		&role,
	)
}

func (u *UserService) RejectUser(
	ctx context.Context,
	tgID int64,
) error {
	return u.repo.UpdateUserStatus(
		ctx,
		tgID,
		"rejected",
		nil,
	)
}

func (u *UserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	user, err := u.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, err
	}
	return user, nil
}

func (u *UserService) AssignSRCode(ctx context.Context, userID string, srCodeID int) error {
	user, err := u.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	if user.Status != "approved" {
		return errors.New("user is not approved")
	}
	if user.Role == nil || *user.Role != "Торговый представитель" {
		return errors.New("user is not a sales representative")
	}
	return u.repo.AssignSRCode(ctx, userID, srCodeID)
}

func (u *UserService) AssignSRCodeAndApprove(ctx context.Context, userID string, srCodeID int, tgID int64) error {
	role := "Торговый Представитель"
	status := "approved"
	user, err := u.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	err = u.repo.AssignSRCodeAndUpdate(ctx, tgID, status, &role, user.ID, srCodeID)
	if err != nil {
		return err
	}
	return nil
}
func (u *UserService) AssignSVTerritoryAndUpdate(ctx context.Context, userID string, territoryID int, tgID int64) error {
	role := "Супервайзер"
	status := "approved"
	user, err := u.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	err = u.repo.AssignSVTerritoryAndUpdate(ctx, tgID, status, &role, user.ID, territoryID)
	if err != nil {
		return err
	}
	return nil
}
