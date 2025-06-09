package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/database/postgres/sqlc"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

const (
	ConfigId = "api"
)

type PostgresRepository struct {
	db      sqlc.DatabaseInterface
	queries sqlc.ConfigQueriesInterface
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

func WithQueriesInterface(queries sqlc.ConfigQueriesInterface) PostgresRepositoryOpt {
	return func(r *PostgresRepository) {
		r.queries = queries
	}
}

func WithDatabaseInterface(db sqlc.DatabaseInterface) PostgresRepositoryOpt {
	return func(r *PostgresRepository) {
		r.db = db
	}
}

// PgxPoolWrapper wraps pgxpool.Pool to implement DatabaseInterface
type PgxPoolWrapper struct {
	*pgxpool.Pool
}

func (w *PgxPoolWrapper) Begin(ctx context.Context) (pgx.Tx, error) {
	return w.Pool.Begin(ctx)
}

// GetUniqueClusterId gets or generates a unique cluster ID
func (r *PostgresRepository) GetUniqueClusterId(ctx context.Context) (string, error) {
	config, err := r.queries.GetConfigByFixedId(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create default config if it doesn't exist
			clusterId := fmt.Sprintf("cluster%s", telemetry.GetMachineID())
			err = r.queries.CreateConfigIfNotExists(ctx, sqlc.CreateConfigIfNotExistsParams{
				ID:              ConfigId,
				ClusterID:       clusterId,
				EnableTelemetry: toPgBool(false),
			})
			if err != nil {
				return "", err
			}
			return clusterId, nil
		}
		return "", err
	}

	// Generate new cluster ID if it's empty
	if config.ClusterID == "" {
		clusterId := fmt.Sprintf("cluster%s", telemetry.GetMachineID())
		err = r.queries.UpdateClusterId(ctx, sqlc.UpdateClusterIdParams{
			ID:        ConfigId,
			ClusterID: clusterId,
		})
		if err != nil {
			return "", err
		}
		return clusterId, nil
	}

	return config.ClusterID, nil
}

// GetTelemetryEnabled returns whether telemetry is enabled
func (r *PostgresRepository) GetTelemetryEnabled(ctx context.Context) (bool, error) {
	enabled, err := r.queries.GetTelemetryEnabled(ctx, ConfigId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create default config if it doesn't exist
			err = r.queries.CreateConfigIfNotExists(ctx, sqlc.CreateConfigIfNotExistsParams{
				ID:              ConfigId,
				ClusterID:       "",
				EnableTelemetry: toPgBool(false),
			})
			if err != nil {
				return false, err
			}
			return false, nil
		}
		return false, err
	}
	return enabled.Bool, nil
}

// Get returns the configuration
func (r *PostgresRepository) Get(ctx context.Context) (testkube.Config, error) {
	config, err := r.queries.GetConfigByFixedId(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return default config if not found
			return testkube.Config{
				Id:              ConfigId,
				ClusterId:       "",
				EnableTelemetry: false,
			}, nil
		}
		return testkube.Config{}, err
	}

	return testkube.Config{
		Id:              config.ID,
		ClusterId:       config.ClusterID,
		EnableTelemetry: config.EnableTelemetry.Bool,
	}, nil
}

// Upsert creates or updates the configuration
func (r *PostgresRepository) Upsert(ctx context.Context, config testkube.Config) (testkube.Config, error) {
	config.Id = ConfigId // Ensure the ID is always "api"

	result, err := r.queries.UpsertConfig(ctx, sqlc.UpsertConfigParams{
		ID:              config.Id,
		ClusterID:       config.ClusterId,
		EnableTelemetry: toPgBool(config.EnableTelemetry),
	})
	if err != nil {
		return testkube.Config{}, err
	}

	return testkube.Config{
		Id:              result.ID,
		ClusterId:       result.ClusterID,
		EnableTelemetry: result.EnableTelemetry.Bool,
	}, nil
}

func toPgBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}
