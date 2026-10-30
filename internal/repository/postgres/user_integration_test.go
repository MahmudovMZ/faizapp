//go:build integration

package postgres

import (
	"context"
	"errors"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// randomTestTgID returns a Telegram ID far above real ones,
// so parallel or repeated runs do not collide.
func randomTestTgID() int64 {
	return 9_000_000_000_000 + rand.Int64N(1_000_000_000)
}

func TestCreatePendingUser(t *testing.T) {
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

	newTgID := randomTestTgID()
	approvedTgID := newTgID + 1

	defer func() {
		_, _ = pool.Exec(
			ctx,
			`DELETE FROM users WHERE tg_id = ANY($1)`,
			[]int64{newTgID, approvedTgID},
		)
	}()

	first, err := repo.CreatePendingUser(ctx, newTgID, "Pending Test User")
	if err != nil {
		t.Fatalf("create pending user: %v", err)
	}

	if first.ID == "" {
		t.Fatal("user ID is empty")
	}
	if first.TgID != newTgID {
		t.Fatalf("tg_id = %d, want %d", first.TgID, newTgID)
	}
	if first.FullName != "Pending Test User" {
		t.Fatalf("full_name = %q, want %q", first.FullName, "Pending Test User")
	}
	if first.Status != "pending" {
		t.Fatalf("status = %q, want pending", first.Status)
	}
	if first.Phone != nil {
		t.Fatalf("phone = %q, want nil", *first.Phone)
	}

	second, err := repo.CreatePendingUser(ctx, newTgID, "Another Name")
	if err != nil {
		t.Fatalf("create pending user again: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("second ID = %s, want %s", second.ID, first.ID)
	}
	if second.FullName != "Pending Test User" {
		t.Fatalf("full_name changed to %q", second.FullName)
	}

	var rows int
	err = pool.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM users WHERE tg_id = $1`,
		newTgID,
	).Scan(&rows)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if rows != 1 {
		t.Fatalf("users with tg_id = %d, want 1", rows)
	}

	var approvedID string
	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, status)
		VALUES ($1, $2, 'approved')
		RETURNING id
		`,
		approvedTgID,
		"Approved Test User",
	).Scan(&approvedID)
	if err != nil {
		t.Fatalf("create approved user: %v", err)
	}

	approved, err := repo.CreatePendingUser(ctx, approvedTgID, "Telegram Name")
	if err != nil {
		t.Fatalf("create pending user for approved: %v", err)
	}

	if approved.ID != approvedID {
		t.Fatalf("approved ID = %s, want %s", approved.ID, approvedID)
	}
	if approved.Status != "approved" {
		t.Fatalf("status = %q, want approved", approved.Status)
	}
	if approved.FullName != "Approved Test User" {
		t.Fatalf("full_name changed to %q", approved.FullName)
	}
}

func TestUpdateUserContact(t *testing.T) {
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

	pendingTgID := randomTestTgID()
	approvedTgID := pendingTgID + 1
	missingTgID := pendingTgID + 2

	defer func() {
		_, _ = pool.Exec(
			ctx,
			`DELETE FROM users WHERE tg_id = ANY($1)`,
			[]int64{pendingTgID, approvedTgID, missingTgID},
		)
	}()

	created, err := repo.CreatePendingUser(ctx, pendingTgID, "Telegram Name")
	if err != nil {
		t.Fatalf("create pending user: %v", err)
	}

	err = repo.UpdateUserContact(ctx, pendingTgID, "Registered Name", "+992900000001")
	if err != nil {
		t.Fatalf("update pending user contact: %v", err)
	}

	updated, err := repo.GetUserByTgID(ctx, pendingTgID)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}

	if updated.ID != created.ID {
		t.Fatalf("ID = %s, want %s", updated.ID, created.ID)
	}
	if updated.FullName != "Registered Name" {
		t.Fatalf("full_name = %q, want %q", updated.FullName, "Registered Name")
	}
	if updated.Phone == nil || *updated.Phone != "+992900000001" {
		t.Fatalf("phone = %v, want +992900000001", updated.Phone)
	}
	if updated.Status != "pending" {
		t.Fatalf("status = %q, want pending", updated.Status)
	}

	_, err = pool.Exec(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, phone, status)
		VALUES ($1, $2, $3, 'approved')
		`,
		approvedTgID,
		"Approved Name",
		"+992900000002",
	)
	if err != nil {
		t.Fatalf("create approved user: %v", err)
	}

	err = repo.UpdateUserContact(ctx, approvedTgID, "Changed Name", "+992900000003")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("approved user: error = %v, want pgx.ErrNoRows", err)
	}

	approved, err := repo.GetUserByTgID(ctx, approvedTgID)
	if err != nil {
		t.Fatalf("get approved user: %v", err)
	}
	if approved.FullName != "Approved Name" {
		t.Fatalf("approved full_name changed to %q", approved.FullName)
	}
	if approved.Phone == nil || *approved.Phone != "+992900000002" {
		t.Fatalf("approved phone changed to %v", approved.Phone)
	}

	err = repo.UpdateUserContact(ctx, missingTgID, "Nobody", "+992900000004")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("missing user: error = %v, want pgx.ErrNoRows", err)
	}
}
