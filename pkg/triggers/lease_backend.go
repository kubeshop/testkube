package triggers

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoCollectionTriggersLease = "triggers"
	documentType                 = "lease"
)

// LeaseBackend does a check and set operation on the Lease object in the defined data source
//
//go:generate mockgen -destination=./mock_lease_backend.go -package=triggers "github.com/kubeshop/testkube/pkg/triggers" LeaseBackend
type LeaseBackend interface {
	// TryAcquire tries to acquire lease from underlying datastore
	TryAcquire(ctx context.Context, id, clusterID string) (leased bool, err error)
}

type AcquireAlwaysLeaseBackend struct{}

func NewAcquireAlwaysLeaseBackend() *AcquireAlwaysLeaseBackend {
	return &AcquireAlwaysLeaseBackend{}
}

func (b *AcquireAlwaysLeaseBackend) TryAcquire(ctx context.Context, id, clusterID string) (leased bool, err error) {
	return true, nil
}

type MongoLeaseBackend struct {
	coll *mongo.Collection
}

func NewMongoLeaseBackend(db *mongo.Database) *MongoLeaseBackend {
	return &MongoLeaseBackend{coll: db.Collection(mongoCollectionTriggersLease)}
}

func (b *MongoLeaseBackend) TryAcquire(ctx context.Context, id, clusterID string) (leased bool, err error) {
	leaseMongoID := newLeaseMongoID(clusterID)
	currentLease, err := b.findOrInsertCurrentLease(ctx, leaseMongoID, id, clusterID)
	if err != nil {
		return false, err
	}

	acquired, renewable := leaseStatus(currentLease, id, clusterID)
	switch {
	case acquired:
		return true, nil
	case !renewable:
		return false, nil
	}

	acquiredAt := currentLease.AcquiredAt
	if currentLease.Identifier != id {
		acquiredAt = time.Now()
	}
	newLease, err := b.tryUpdateLease(ctx, leaseMongoID, id, clusterID, acquiredAt)
	if err != nil {
		return false, err
	}
	acquired, _ = leaseStatus(newLease, id, clusterID)

	return acquired, nil
}

func (b *MongoLeaseBackend) findOrInsertCurrentLease(ctx context.Context, leaseMongoID, id, clusterID string) (*Lease, error) {
	res := b.coll.FindOne(ctx, bson.M{"_id": leaseMongoID})
	if res.Err() == mongo.ErrNoDocuments {
		lease, err := b.insertLease(ctx, leaseMongoID, id, clusterID)
		if err != nil {
			return nil, err
		}
		return lease, err
	} else if res.Err() != nil {
		return nil, errors.Wrap(res.Err(), "error finding lease document in mongo")
	}

	var receivedLease MongoLease
	if err := res.Decode(&receivedLease); err != nil {
		return nil, errors.Wrap(err, "error decoding lease mongo document")
	}

	return &receivedLease.Lease, nil
}

func (b *MongoLeaseBackend) insertLease(ctx context.Context, leaseMongoID, id, clusterID string) (*Lease, error) {
	lease := NewLease(id, clusterID)
	_, err := b.coll.InsertOne(ctx, bson.M{"_id": leaseMongoID, "lease": *lease})
	if err != nil {
		return nil, errors.Wrap(err, "error inserting lease document into mongo")
	}
	return lease, nil
}

func (b *MongoLeaseBackend) tryUpdateLease(ctx context.Context, leaseMongoID, id, clusterID string, acquiredAt time.Time) (*Lease, error) {
	newLease := Lease{
		Identifier: id,
		ClusterID:  clusterID,
		AcquiredAt: acquiredAt,
		RenewedAt:  time.Now(),
	}
	newMongoLease := MongoLease{
		_id:   leaseMongoID,
		Lease: newLease,
	}

	after := options.After
	opts := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
	}
	res := b.coll.FindOneAndUpdate(
		ctx,
		bson.M{"_id": leaseMongoID},
		bson.M{"$set": newMongoLease},
		&opts,
	)
	if res.Err() != nil {
		return nil, errors.Wrap(res.Err(), "error finding and updating mongo db document")
	}
	var updatedLease MongoLease
	if err := res.Decode(&updatedLease); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling returned lease mongo document")
	}

	return &updatedLease.Lease, nil
}

func leaseStatus(lease *Lease, id, clusterID string) (acquired bool, renewable bool) {
	if lease == nil {
		return false, false
	}
	maxLeaseDurationStaleness := time.Now().Add(-defaultMaxLeaseDuration)
	isLeaseExpired := lease.RenewedAt.Before(maxLeaseDurationStaleness)
	isMyLease := lease.Identifier == id && lease.ClusterID == clusterID
	switch {
	case isLeaseExpired:
		acquired = false
		renewable = true
	case isMyLease:
		acquired = true
		renewable = false
	default:
		acquired = false
		renewable = false
	}
	return
}

func newLeaseMongoID(clusterID string) string {
	return fmt.Sprintf("%s-%s", documentType, clusterID)
}

type MongoLease struct {
	_id string `bson:"_id"`
	Lease
}
