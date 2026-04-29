package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlc "github.com/kubeshop/testkube/pkg/controlplane/scheduling/sqlc"
	"github.com/kubeshop/testkube/pkg/log"
)

type DB struct {
	Pool *pgxpool.Pool
	*sqlc.Queries
}

func CreateDatabaseIfNotExists(ctx context.Context, connectionString string) error {
	connConfig, err := pgx.ParseConfig(connectionString)
	if err != nil {
		return err
	}
	log.DefaultLogger.Infof("Attempting to create database %q on host %s", connConfig.Database, connConfig.Host)

	if connConfig.Database == "" {
		log.DefaultLogger.Info("No database in connection string")
		return nil
	}

	dbName := connConfig.Database
	connConfig.Database = "postgres"

	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{dbName}.Sanitize())
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P04" {
			log.DefaultLogger.Infof("Database %s already exists", dbName)
		} else {
			return err
		}
	} else {
		log.DefaultLogger.Infof("Database %s created successfully", dbName)
	}

	return nil
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
