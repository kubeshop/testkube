package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDatabaseIfNotExists_EmptyDatabase(t *testing.T) {
	// When no database name is in the connection string, it should be a no-op.
	err := CreateDatabaseIfNotExists(context.Background(), "postgres://user:pass@localhost:5432/")
	assert.NoError(t, err)
}

func TestCreateDatabaseIfNotExists_InvalidConnectionString(t *testing.T) {
	// Invalid connection strings should return an error.
	err := CreateDatabaseIfNotExists(context.Background(), "not-a-valid-dsn://???")
	assert.Error(t, err)
}
