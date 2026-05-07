package mongo

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const defaultMongoTestURL = "mongodb://localhost:27017"

func PrepareMongoTestDatabase(t *testing.T, baseName string) (*mongo.Database, func()) {
	t.Helper()

	ctx := context.Background()
	uri := os.Getenv("MONGO_URL")
	if uri == "" {
		uri = defaultMongoTestURL
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)

	dbName := generateTestDBName(baseName, t.Name())
	db := client.Database(dbName)

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			_ = db.Drop(ctx)
			_ = client.Disconnect(ctx)
		})
	}

	t.Cleanup(cleanup)

	return db, cleanup
}

func generateTestDBName(baseName, testName string) string {
	const hashLen = 12
	seed := fmt.Sprintf("%s_%s_%d", baseName, testName, time.Now().UnixNano())
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(seed)))[:hashLen]
	sanitized := sanitizeIdentifier(baseName)
	return fmt.Sprintf("%s_%s", sanitized, hash)
}

func sanitizeIdentifier(name string) string {
	var builder []byte
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			builder = append(builder, byte(r))
		case r >= '0' && r <= '9':
			builder = append(builder, byte(r))
		case r == '_':
			builder = append(builder, byte(r))
		default:
			builder = append(builder, '_')
		}
	}
	result := string(builder)
	if result == "" {
		result = "testdb"
	}
	return result
}
