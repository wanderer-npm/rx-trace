package db

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://rx:rx@localhost:5432/rxtrace?sslmode=disable"
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnIdleTime = 5 * time.Minute
	return pgxpool.NewWithConfig(ctx, cfg)
}
