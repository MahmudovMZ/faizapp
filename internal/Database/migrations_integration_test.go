//go:build integration

package Database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationsAndSeedData(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	root := findRepoRoot(t)

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// Повторный запуск не должен ломаться.
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("repeat migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	assertCountAtLeast(t, ctx, pool, "work_groups", 1)
	assertCountAtLeast(t, ctx, pool, "territories", 1)
	assertCountAtLeast(t, ctx, pool, "brand_categories", 1)
	assertCountAtLeast(t, ctx, pool, "sr_codes", 1)

	var sahoCount int
	err = pool.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM sr_codes
		 WHERE code IN ('SAHO-1', 'SAHO-2', 'SAHO-3', 'SAHO-4')`,
	).Scan(&sahoCount)
	if err != nil {
		t.Fatalf("check SAHO codes: %v", err)
	}

	if sahoCount != 4 {
		t.Fatalf("SAHO codes count = %d, want 4", sahoCount)
	}
}

func assertCountAtLeast(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
	want int,
) {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM " + table

	if err := pool.QueryRow(ctx, query).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	if count < want {
		t.Fatalf("%s count = %d, want at least %d", table, count, want)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}

		dir = parent
	}
}
