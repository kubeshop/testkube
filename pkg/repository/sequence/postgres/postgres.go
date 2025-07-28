package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
)

type PostgresRepository struct {
	db      sqlc.DatabaseInterface
	queries sqlc.ExecutionSequenceQueriesInterface
}

type PostgresRepositoryOpt func(*PostgresRepository)

func NewPostgresRepository(db *pgxpool.Pool, opts ...PostgresRepositoryOpt) *PostgresRepository {
	r := &PostgresRepository{
		db:      &PgxPoolWrapper{Pool: db},
		queries: sqlc.New(db),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func WithQueriesInterface(queries sqlc.ExecutionSequenceQueriesInterface) PostgresRepositoryOpt {
	return func(r *PostgresRepository) {
		r.queries = queries
	}
}

func WithDatabaseInterface(db sqlc.DatabaseInterface) PostgresRepositoryOpt {
	return func(b *PostgresRepository) {
		b.db = db
	}
}

// PgxPoolWrapper wraps pgxpool.Pool to implement DatabaseInterface
type PgxPoolWrapper struct {
	*pgxpool.Pool
}

func (w *PgxPoolWrapper) Begin(ctx context.Context) (pgx.Tx, error) {
	return w.Pool.Begin(ctx)
}

// GetNextExecutionNumber gets next execution number by name using atomic upsert
func (r *PostgresRepository) GetNextExecutionNumber(ctx context.Context, name string, _ sequence.ExecutionType) (int32, error) {
	result, err := r.queries.UpsertAndIncrementExecutionSequence(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("failed to get next execution number: %w", err)
	}

	return int32(result.Number), nil
}

// DeleteExecutionNumber deletes execution number by name
func (r *PostgresRepository) DeleteExecutionNumber(ctx context.Context, name string, _ sequence.ExecutionType) error {
	err := r.queries.DeleteExecutionSequence(ctx, name)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to delete execution sequence: %w", err)
	}
	return nil
}

// DeleteExecutionNumbers deletes multiple execution numbers by names
func (r *PostgresRepository) DeleteExecutionNumbers(ctx context.Context, names []string, _ sequence.ExecutionType) error {
	if len(names) == 0 {
		return nil
	}

	err := r.queries.DeleteExecutionSequences(ctx, names)
	if err != nil {
		return fmt.Errorf("failed to delete execution sequences: %w", err)
	}

	return nil
}

// DeleteAllExecutionNumbers deletes all execution numbers
func (r *PostgresRepository) DeleteAllExecutionNumbers(ctx context.Context, _ sequence.ExecutionType) error {
	err := r.queries.DeleteAllExecutionSequences(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete all execution sequences: %w", err)
	}
	return nil
}
