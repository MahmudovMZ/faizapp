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
	AssignSRCode(ctx context.Context, userID string, srCodeID int) error
	GetAssignmentsByUser(ctx context.Context, userID string) ([]*models.SRAssignment, error)
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	GetWorkGroups(ctx context.Context) ([]models.WorkGroup, error)
	GetTerritoriesByGroup(ctx context.Context, workGroupID int) ([]models.Territory, error)
	GetAvailableSRCodes(ctx context.Context, territoryID int) ([]models.SRCode, error)
	AssignSRCodeAndUpdate(ctx context.Context, tgID int64, status string, role *string, userID string, srCodeID int) error
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

func (r *FaizAppRepo) AssignSRCodeAndUpdate(
	ctx context.Context,
	tgID int64,
	status string,
	role *string,
	userID string,
	srCodeID int,
) error {
	log.Println("[REPOSITORY] assigning SR code and updating user")

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	insertQuery := `
		INSERT INTO sr_assignments (user_id, sr_code_id)
		VALUES ($1, $2)
	`

	if _, err := tx.Exec(ctx, insertQuery, userID, srCodeID); err != nil {
		return err
	}

	updateQuery := `
		UPDATE users
		SET status = $1,
		    role = $2
		WHERE id = $3
		  AND tg_id = $4
		  AND status = 'pending'
	`

	result, err := tx.Exec(
		ctx,
		updateQuery,
		status,
		role,
		userID,
		tgID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() != 1 {
		return fmt.Errorf("user not found or already processed")
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
func (r *FaizAppRepo) GetAssignmentsByUser(ctx context.Context, userID string) ([]*models.SRAssignment, error) {
	log.Println("[REPOSITORY] getting assignments")
	var assignments []*models.SRAssignment

	query := `SELECT id, sr_code_id, assigned_at FROM sr_assignments WHERE user_id = $1 ORDER BY assigned_at`
	rows, err := r.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		assignment := &models.SRAssignment{
			UserID: userID,
		}

		if err := rows.Scan(
			&assignment.ID,
			&assignment.SRCodeID,
			&assignment.AssignedAt); err != nil {
			return nil, err
		}

		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return assignments, nil
}
func (r *FaizAppRepo) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	log.Println("[REPOSITORY] getting user by id")
	var user models.User
	query := `SELECT id, tg_id,full_name,phone,role,status,created_at FROM users WHERE id = $1`
	err := r.Pool.QueryRow(ctx, query, userID).Scan(
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

func (r *FaizAppRepo) AssignSRCode(
	ctx context.Context,
	userID string,
	srCodeID int,
) error {
	log.Println("[REPOSITORY] assigning SR code")

	query := `
		INSERT INTO sr_assignments (user_id, sr_code_id)
		VALUES ($1, $2)
	`

	_, err := r.Pool.Exec(ctx, query, userID, srCodeID)
	return err
}
