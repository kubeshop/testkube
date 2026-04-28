package postgres

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	database "github.com/kubeshop/testkube/pkg/database/postgres"
	"github.com/kubeshop/testkube/pkg/database/postgres/migrations"
)

const defaultPostgresTestURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

// PreparePostgresTestDatabase creates a temporary Postgres database with migrations applied.
func PreparePostgresTestDatabase(t *testing.T, baseName string) (*database.DB, func()) {
	t.Helper()

	ctx := context.Background()
	dsn := os.Getenv("API_POSTGRES_URL")
	if dsn == "" {
		dsn = defaultPostgresTestURL
	}

	// Parse the connection config
	poolCfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)

	// Connect to the base database to create our test database
	basePool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	// Generate unique database name
	dbName := generateTestDBName(baseName, t.Name())

	// Create fresh test database
	_, err = basePool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	require.NoError(t, err)
	_, err = basePool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE template0", dbName))
	require.NoError(t, err)

	// Connect to the new database by modifying the parsed config
	poolCfg.ConnConfig.Database = dbName
	testPool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	require.NoError(t, err)

	testDB := &database.DB{
		Pool: testPool,
	}

	// Run migrations
	sqlDB := stdlib.OpenDBFromPool(testPool)
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.Fs)
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			testDB.Close()
			if _, err := basePool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)); err != nil {
				t.Logf("failed to drop database %s: %v", dbName, err)
			}
			basePool.Close()
		})
	}

	t.Cleanup(cleanup)

	return testDB, cleanup
}

// generateTestDBName creates a unique, valid Postgres database name.
func generateTestDBName(baseName, testName string) string {
	sanitizedBase := SanitizeIdentifier(baseName)
	const hashLen = 12
	const maxIdentifierLen = 63
	maxBaseLen := maxIdentifierLen - hashLen - 1
	if maxBaseLen < 1 {
		maxBaseLen = 1
	}
	if len(sanitizedBase) > maxBaseLen {
		sanitizedBase = sanitizedBase[:maxBaseLen]
	}
	seed := fmt.Sprintf("%s_%s_%d", baseName, testName, time.Now().UnixNano())
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(seed)))[:hashLen]
	return fmt.Sprintf("%s_%s", sanitizedBase, hash)
}

// SanitizeIdentifier sanitizes a name to be a valid Postgres identifier.
func SanitizeIdentifier(name string) string {
	name = strings.ToLower(name)
	var builder strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	sanitized := builder.String()
	if sanitized == "" {
		sanitized = "testdb"
	}
	if sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "db_" + sanitized
	}

	return sanitized
}
