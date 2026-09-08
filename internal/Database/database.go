package Database

import (
	"context"
	"fmt"
	"log"

	"github.com/MahmudovMZ/faizapp/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	log.Println("[DATABASE] Initializing database...")
	dsn := BuildDSN(cfg)

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	log.Println("[DATABASE] Connected to database")
	return db, nil
}

func BuildDSN(cfg config.DBConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%v/%s?sslmode=disable",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
