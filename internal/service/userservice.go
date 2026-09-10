package service

import (
	"context"
	"errors"
	"fmt"

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
