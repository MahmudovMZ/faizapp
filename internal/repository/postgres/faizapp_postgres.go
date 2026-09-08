package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type FaizAppRepo struct {
	Pool *pgxpool.Pool
}

func NewFaizAppRepo(pool *pgxpool.Pool) FaizAppRepo {
	return FaizAppRepo{
		Pool: pool,
	}
}
