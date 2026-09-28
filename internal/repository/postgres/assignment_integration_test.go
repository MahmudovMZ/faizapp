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
