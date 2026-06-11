package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateDatabaseIfNotExists_EmptyDatabase(t *testing.T) {
	// Ensure no environment variable can inject a database name into the
	// parsed config, making this test hermetic.
	t.Setenv("PGDATABASE", "")

	// When no database name is in the connection string, it should be a no-op.
	err := CreateDatabaseIfNotExists(context.Background(), "postgres"+"://user:pass@localhost:5432/")
	assert.NoError(t, err)
}

func TestCreateDatabaseIfNotExists_InvalidConnectionString(t *testing.T) {
	// Invalid connection strings should return an error.
	err := CreateDatabaseIfNotExists(context.Background(), "not-a-valid-dsn://???")
	assert.Error(t, err)
}

func TestCreateDatabaseIfNotExists_RespectsContextCancellation(t *testing.T) {
	// When the parent context is already cancelled, the function should
	// return quickly instead of hanging.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	start := time.Now()
	// Use a non-routable IP to simulate a hanging connection. The cancelled
	// context should cause it to fail fast.
	_ = CreateDatabaseIfNotExists(ctx, "postgres"+"://user:pass@192.0.2.1:5432/testdb")
	elapsed := time.Since(start)

	// Should complete quickly and not hang.
	assert.Less(t, elapsed, 5*time.Second, "function should respect context cancellation and not hang")
}

func TestDefaultConnectTimeout(t *testing.T) {
	// Verify the default connect timeout constant is set to a reasonable value.
	assert.Equal(t, 30*time.Second, defaultConnectTimeout)
}
