//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAssignSRCodeAndUpdateRollback(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := NewFaizAppRepo(pool)

	var userID string
	const tgID int64 = 999999991

	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, status)
		VALUES ($1, $2, 'pending')
		RETURNING id
		`,
		tgID,
		"Integration Test User",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			ctx,
			`DELETE FROM sr_assignments WHERE user_id = $1`,
			userID,
		)
		_, _ = pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	var srCodeID int

	err = pool.QueryRow(
		ctx,
		`
		SELECT s.id
		FROM sr_codes s
		LEFT JOIN sr_assignments a
			ON a.sr_code_id = s.id
		WHERE a.sr_code_id IS NULL
		ORDER BY s.id
		LIMIT 1
		`,
	).Scan(&srCodeID)
	if err != nil {
		t.Fatalf("find free sr code: %v", err)
	}

	role := "Торговый Представитель"

	err = repo.AssignSRCodeAndUpdate(
		ctx,
		999999992, // неправильный tgID
		"approved",
		&role,
		userID,
		srCodeID,
	)
	if err == nil {
		t.Fatal("expected update error, got nil")
	}

	var assignments int
	err = pool.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM sr_assignments
		WHERE user_id = $1
		  AND sr_code_id = $2
		`,
		userID,
		srCodeID,
	).Scan(&assignments)
	if err != nil {
		t.Fatalf("check rollback: %v", err)
	}

	if assignments != 0 {
		t.Fatalf("assignment count = %d, want 0", assignments)
	}
}
func TestAssignSVWorkGroupAndUpdate(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := NewFaizAppRepo(pool)

	const tgID int64 = 999999993

	var userID string
	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, status)
		VALUES ($1, $2, 'pending')
		RETURNING id
		`,
		tgID,
		"SV Integration Test",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	var workGroupID int
	err = pool.QueryRow(
		ctx,
		`
		SELECT id
		FROM work_groups
		WHERE supervisor_id IS NULL
		ORDER BY id
		LIMIT 1
		`,
	).Scan(&workGroupID)
	if err != nil {
		t.Fatalf("find free work group: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			ctx,
			`UPDATE work_groups SET supervisor_id = NULL WHERE id = $1`,
			workGroupID,
		)
		_, _ = pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	role := "Супервайзер"

	err = repo.AssignSVWorkGroupAndUpdate(
		ctx,
		tgID,
		"approved",
		&role,
		userID,
		workGroupID,
	)
	if err != nil {
		t.Fatalf("assign supervisor: %v", err)
	}

	var supervisorID *string
	err = pool.QueryRow(
		ctx,
		`SELECT supervisor_id FROM work_groups WHERE id = $1`,
		workGroupID,
	).Scan(&supervisorID)
	if err != nil {
		t.Fatalf("check supervisor: %v", err)
	}

	if supervisorID == nil || *supervisorID != userID {
		t.Fatalf("supervisor_id = %v, want %s", supervisorID, userID)
	}

	var status string
	var savedRole *string

	err = pool.QueryRow(
		ctx,
		`SELECT status, role FROM users WHERE id = $1`,
		userID,
	).Scan(&status, &savedRole)
	if err != nil {
		t.Fatalf("check user: %v", err)
	}

	if status != "approved" {
		t.Fatalf("status = %s, want approved", status)
	}

	if savedRole == nil || *savedRole != role {
		t.Fatalf("role = %v, want %s", savedRole, role)
	}
}
