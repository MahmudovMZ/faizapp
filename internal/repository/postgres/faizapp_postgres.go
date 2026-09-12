package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByTgID(ctx context.Context, tgId int64) (*models.User, error)
	UpdateUserStatus(ctx context.Context, tgID int64, status string, role *string) error
}

type FaizAppRepo struct {
	Pool *pgxpool.Pool
}

func NewFaizAppRepo(pool *pgxpool.Pool) *FaizAppRepo {
	return &FaizAppRepo{
		Pool: pool,
	}
}

func (r *FaizAppRepo) CreateUser(ctx context.Context, user *models.User) error {
	log.Println("[REPOSITORY] creating user")
	query := `INSERT INTO users (tg_id, full_name, phone) values ($1, $2, $3)`

	_, err := r.Pool.Exec(ctx, query, user.TgID, user.FullName, user.Phone)
	if err != nil {
		return err
	}
	return nil
}

func (r *FaizAppRepo) GetUserByTgID(ctx context.Context, tgId int64) (*models.User, error) {
	log.Println("[REPOSITORY] getting user by tgId")
	var user models.User
	query := `SELECT id,tg_id,full_name,phone,role,status,created_at FROM users WHERE tg_id = $1`
	err := r.Pool.QueryRow(ctx, query, tgId).Scan(
		&user.ID,
		&user.TgID,
		&user.FullName,
		&user.Phone,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *FaizAppRepo) UpdateUserStatus(ctx context.Context, tgID int64, status string, role *string) error {
	log.Println("[REPOSITORY] updating user")

	query := `
      UPDATE users
SET status = $1,
    role = $2
WHERE tg_id = $3
  AND status = 'pending'
    `

	result, err := r.Pool.Exec(
		ctx,
		query,
		status,
		role,
		tgID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already processed")
	}

	return nil
}
