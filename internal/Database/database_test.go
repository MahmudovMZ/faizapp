package Database

import (
	"context"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/config"
)

func TestNewPoolWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := config.DBConfig{
		Username: "test_user",
		Password: "test_password",
		DBName:   "test_db",
		Host:     "localhost",
		Port:     5432,
	}

	pool, err := NewPool(ctx, cfg)

	if err == nil {
		t.Fatal("expected error when context is canceled")
	}

	if pool != nil {
		t.Fatal("expected nil pool when connection fails")
	}
}

func TestNewPoolWithUnavailableDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()

	cfg := config.DBConfig{
		Username: "test_user",
		Password: "test_password",
		DBName:   "test_db",
		Host:     "127.0.0.1",
		Port:     1,
	}

	pool, err := NewPool(ctx, cfg)

	if err == nil {
		t.Fatal("expected error when database is unavailable")
	}

	if pool != nil {
		t.Fatal("expected nil pool when database is unavailable")
	}
}
