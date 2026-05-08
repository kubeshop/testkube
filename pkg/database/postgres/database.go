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

// CreateDatabaseIfNotExists ensures the database specified in the connection
// string exists. It first attempts a direct connection to the target database;
// if that succeeds the database already exists and no further action is taken.
// Only when the target database is unreachable does it fall back to connecting
// to the "postgres" system database and issuing CREATE DATABASE. This ordering
// avoids failures in managed/cloud environments where restricted users cannot
// connect to the "postgres" database at all.
func CreateDatabaseIfNotExists(ctx context.Context, connectionString string) error {
	connConfig, err := pgx.ParseConfig(connectionString)
	if err != nil {
		return err
	}

	if connConfig.Database == "" {
		log.DefaultLogger.Info("No database in connection string, skipping creation check")
		return nil
	}

	dbName := connConfig.Database
	log.DefaultLogger.Infof("Checking if database %q exists on host %s", dbName, connConfig.Host)

	// Fast path: try to connect directly to the target database.
	// If successful the database already exists — nothing to do.
	directConn, err := pgx.ConnectConfig(ctx, connConfig)
	if err == nil {
		directConn.Close(ctx)
		log.DefaultLogger.Infof("Database %s already exists", dbName)
		return nil
	}

	// The direct connection failed. Attempt to create the database by
	// connecting to the "postgres" system database.
	log.DefaultLogger.Infof("Cannot connect to database %s directly (%v), attempting to create it", dbName, err)

	connConfig.Database = "postgres"
	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		// Cannot reach the system database either. The target database may
		// actually exist but we simply cannot verify it from here. Return nil
		// and let the normal pool connection (which uses the original DSN)
		// surface a clear error if the database truly does not exist.
		log.DefaultLogger.Warnf("Cannot connect to system database to create %s: %v; will attempt direct connection later", dbName, err)
		return nil
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
