package triggers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestMongoLeaseBackend_TryAcquire(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("acquire existing lease", func(mt *mtest.T) {
		mt.Parallel()

		leaseBackend := NewMongoLeaseBackend(mt.DB)

		testID := "test-host-1"
		testClusterID := "testkube_api"
		expectedLease := Lease{
			Identifier: testID,
			ClusterID:  testClusterID,
			AcquiredAt: time.Now(),
			RenewedAt:  time.Now(),
		}
		expectedMongoLease := MongoLease{
			_id:   newLeaseMongoID(testClusterID),
			Lease: expectedLease,
		}

		bsonD := cycleBSON(&expectedMongoLease)

		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"testkube.lease",
			mtest.FirstBatch,
			bsonD,
		))

		leased, err := leaseBackend.TryAcquire(ctx, testID, testClusterID)

		assert.True(t, leased, "should acquire lease")
		assert.NoError(t, err)
	})

	mt.Run("renew existing lease", func(mt *mtest.T) {
		mt.Parallel()

		leaseBackend := NewMongoLeaseBackend(mt.DB)

		testID := "test-host-1"
		testClusterID := "testkube_api"
		acquiredAt := time.Now().Add(-1 * time.Hour)
		expectedLease1 := Lease{
			Identifier: testID,
			ClusterID:  testClusterID,
			AcquiredAt: acquiredAt,
			RenewedAt:  time.Now().Add(-1 * time.Hour),
		}
		expectedMongoLease1 := MongoLease{
			_id:   newLeaseMongoID(testClusterID),
			Lease: expectedLease1,
		}
		expectedLease2 := Lease{
			Identifier: testID,
			ClusterID:  testClusterID,
			AcquiredAt: acquiredAt,
			RenewedAt:  time.Now(),
		}
		expectedMongoLease2 := MongoLease{
			_id:   newLeaseMongoID(testClusterID),
			Lease: expectedLease2,
		}

		bsonD1 := cycleBSON(&expectedMongoLease1)
		bsonD2 := cycleBSON(&expectedMongoLease2)

		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"testkube.lease",
			mtest.FirstBatch,
			bsonD1,
		))
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testkube.lease",
			mtest.FirstBatch,
		))
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "value", Value: bsonD2},
		})

		leased, err := leaseBackend.TryAcquire(ctx, testID, testClusterID)

		assert.True(t, leased, "should acquire lease")
		assert.NoError(t, err)
	})

	mt.Run("not acquire if other instance is holding non-expired", func(mt *mtest.T) {
		mt.Parallel()

		leaseBackend := NewMongoLeaseBackend(mt.DB)

		testClusterID := "testkube_api"
		acquiredAt := time.Now().Add(-1 * time.Hour)
		expectedLease := Lease{
			Identifier: "test-id-2",
			ClusterID:  testClusterID,
			AcquiredAt: acquiredAt,
			RenewedAt:  time.Now().Add(-5 * time.Second),
		}
		expectedMongoLease := MongoLease{
			_id:   newLeaseMongoID(testClusterID),
			Lease: expectedLease,
		}

		bsonD := cycleBSON(&expectedMongoLease)

		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"testkube.lease",
			mtest.FirstBatch,
			bsonD,
		))

		leased, err := leaseBackend.TryAcquire(ctx, "test-host-1", testClusterID)

		assert.False(t, leased, "should not acquire lease")
		assert.NoError(t, err)
	})

	mt.Run("acquire lease from other instance if lease is expired", func(mt *mtest.T) {
		mt.Parallel()

		leaseBackend := NewMongoLeaseBackend(mt.DB)

		testClusterID := "testkube_api"
		acquiredAt := time.Now().Add(-1 * time.Hour)
		expectedLease1 := Lease{
			Identifier: "test-host-2",
			ClusterID:  testClusterID,
			AcquiredAt: acquiredAt,
			RenewedAt:  time.Now().Add(-1 * time.Hour),
		}
		expectedMongoLease1 := MongoLease{
			_id:   newLeaseMongoID(testClusterID),
			Lease: expectedLease1,
		}
		expectedLease2 := Lease{
			Identifier: "test-host-1",
			ClusterID:  testClusterID,
			AcquiredAt: acquiredAt,
			RenewedAt:  time.Now(),
		}
		expectedMongoLease2 := MongoLease{
			_id:   newLeaseMongoID(testClusterID),
			Lease: expectedLease2,
		}

		bsonD1 := cycleBSON(&expectedMongoLease1)
		bsonD2 := cycleBSON(&expectedMongoLease2)

		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"testkube.lease",
			mtest.FirstBatch,
			bsonD1,
		))
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testkube.lease",
			mtest.FirstBatch,
		))
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "value", Value: bsonD2},
		})

		leased, err := leaseBackend.TryAcquire(ctx, "test-host-1", testClusterID)

		assert.True(t, leased, "should acquire lease")
		assert.NoError(t, err)
	})
}

func cycleBSON(data any) bson.D {
	bsonData, _ := bson.Marshal(data)
	var bsonD bson.D
	_ = bson.Unmarshal(bsonData, &bsonD)

	return bsonD
}
