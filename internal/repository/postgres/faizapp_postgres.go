package postgres

import (
	"context"
	"log"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByTgID(ctx context.Context, tgId string) (*models.User, error)
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
	query := `INSERT INTO users (tg_id, full_name, phone, role) values ($1, $2, $3, $4)`

	_, err := r.Pool.Exec(ctx, query, user.TgID, user.FullName, user.Phone, user.Role)
	if err != nil {
		return err
	}
	return nil
}

func (r *FaizAppRepo) GetUserByTgID(ctx context.Context, tgId string) (*models.User, error) {
	log.Println("[REPOSITORY] getting user by tgId")
	var user models.User
	query := `SELECT id,tg_id,full_name,phone,role,created_at FROM users WHERE tg_id = $1`
	err := r.Pool.QueryRow(ctx, query, tgId).Scan(
		&user.ID,
		&user.TgID,
		&user.FullName,
		&user.Phone,
		&user.Role,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
