package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlc "github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
)

type DB struct {
	Pool *pgxpool.Pool
	*sqlc.Queries
}

func NewForScheduler(pool *pgxpool.Pool) (*DB, error) {
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

func (db *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.Pool.Begin(ctx)
}
