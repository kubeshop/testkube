package mongo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestMongoLeaseBackend_TryAcquire_Integration(t *testing.T) {
	test.IntegrationTest(t)

	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	t.Cleanup(func() { client.Disconnect(ctx) })

	t.Run("acquire existing lease", func(t *testing.T) {
		db := client.Database("leasebackend-test-acquire-existing")
		t.Cleanup(func() { db.Drop(ctx) })

		leaseBackend := NewMongoLeaseBackend(db)

		testID := "test-host-1"
		testClusterID := "testkube_api"
		leaseMongoID := newLeaseMongoID(testClusterID)

		_, err = db.Collection(mongoCollectionTriggersLease).InsertOne(ctx, bson.M{
			"_id":         leaseMongoID,
			"identifier":  testID,
			"cluster_id":  testClusterID,
			"acquired_at": time.Now(),
			"renewed_at":  time.Now(),
		})
		require.NoError(t, err)

		leased, err := leaseBackend.TryAcquire(ctx, testID, testClusterID)

		assert.True(t, leased, "should acquire lease")
		assert.NoError(t, err)
	})

	t.Run("renew existing lease", func(t *testing.T) {
		db := client.Database("leasebackend-test-renew-existing")
		t.Cleanup(func() { db.Drop(ctx) })

		leaseBackend := NewMongoLeaseBackend(db)

		testID := "test-host-1"
		testClusterID := "testkube_api"
		leaseMongoID := newLeaseMongoID(testClusterID)
		acquiredAt := time.Now().Add(-1 * time.Hour)

		_, err = db.Collection(mongoCollectionTriggersLease).InsertOne(ctx, bson.M{
			"_id":         leaseMongoID,
			"identifier":  testID,
			"cluster_id":  testClusterID,
			"acquired_at": acquiredAt,
			"renewed_at":  time.Now().Add(-1 * time.Hour),
		})
		require.NoError(t, err)

		leased, err := leaseBackend.TryAcquire(ctx, testID, testClusterID)

		assert.True(t, leased, "should acquire lease")
		assert.NoError(t, err)
	})

	t.Run("not acquire if other instance is holding non-expired", func(t *testing.T) {
		db := client.Database("leasebackend-test-not-acquire")
		t.Cleanup(func() { db.Drop(ctx) })

		leaseBackend := NewMongoLeaseBackend(db)

		testClusterID := "testkube_api"
		leaseMongoID := newLeaseMongoID(testClusterID)
		acquiredAt := time.Now().Add(-1 * time.Hour)

		_, err = db.Collection(mongoCollectionTriggersLease).InsertOne(ctx, bson.M{
			"_id":         leaseMongoID,
			"identifier":  "test-id-2",
			"cluster_id":  testClusterID,
			"acquired_at": acquiredAt,
			"renewed_at":  time.Now().Add(-5 * time.Second),
		})
		require.NoError(t, err)

		leased, err := leaseBackend.TryAcquire(ctx, "test-host-1", testClusterID)

		assert.False(t, leased, "should not acquire lease")
		assert.NoError(t, err)
	})

	t.Run("acquire lease from other instance if lease is expired", func(t *testing.T) {
		db := client.Database("leasebackend-test-acquire-expired")
		t.Cleanup(func() { db.Drop(ctx) })

		leaseBackend := NewMongoLeaseBackend(db)

		testClusterID := "testkube_api"
		leaseMongoID := newLeaseMongoID(testClusterID)
		acquiredAt := time.Now().Add(-1 * time.Hour)

		_, err = db.Collection(mongoCollectionTriggersLease).InsertOne(ctx, bson.M{
			"_id":         leaseMongoID,
			"identifier":  "test-host-2",
			"cluster_id":  testClusterID,
			"acquired_at": acquiredAt,
			"renewed_at":  time.Now().Add(-1 * time.Hour),
		})
		require.NoError(t, err)

		leased, err := leaseBackend.TryAcquire(ctx, "test-host-1", testClusterID)

		assert.True(t, leased, "should acquire lease")
		assert.NoError(t, err)
	})
}
