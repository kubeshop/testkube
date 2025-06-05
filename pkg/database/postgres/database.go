package database

import (
	"context"

	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
	*sqlc.Queries
}

func New(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}

	queries := sqlc.New(pool)

	database := DB{
		Queries: queries,
		Pool:    pool,
	}
	return &database, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}
