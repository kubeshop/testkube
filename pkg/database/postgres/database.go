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

// CreateDatabaseIfNotExists attempts to create the database specified in the
// connection string. It is a no-op when the database already exists. If the
// connected user lacks CREATE DATABASE privileges but the target database
// already exists, the error is ignored so that restricted users can still
// start the API successfully.
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
			// 42P04 = duplicate_database: the database already exists
			log.DefaultLogger.Infof("Database %s already exists", dbName)
		} else if errors.As(err, &pgErr) && pgErr.Code == "42501" {
			// 42501 = insufficient_privilege: check whether the DB exists
			exists, checkErr := databaseExists(ctx, conn, dbName)
			if checkErr != nil {
				return err // return original permission error
			}
			if exists {
				log.DefaultLogger.Infof("Database %s already exists (no CREATE privilege, skipping)", dbName)
			} else {
				return err
			}
		} else {
			return err
		}
	} else {
		log.DefaultLogger.Infof("Database %s created successfully", dbName)
	}

	return nil
}

// databaseExists checks whether a database with the given name exists by
// querying the pg_database catalog.
func databaseExists(ctx context.Context, conn *pgx.Conn, dbName string) (bool, error) {
	var exists bool
	err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	return exists, err
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
