package triggers

import (
	"context"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LeaseBackend does a check and set operation on the Lease object in the defined data source
//
//go:generate mockgen -destination=./mock_lease_backend.go -package=triggers "github.com/kubeshop/testkube/pkg/triggers" LeaseBackend
type LeaseBackend interface {
	// CheckAndSet should do a check and set operation with upsert
	CheckAndSet(ctx context.Context, id string) (*Lease, error)
}

type MongoLeaseBackend struct {
	coll *mongo.Collection
}

func NewMongoLeaseBackend(db *mongo.Database) *MongoLeaseBackend {
	return &MongoLeaseBackend{coll: db.Collection(mongoCollectionTriggersLease)}
}

func (b *MongoLeaseBackend) CheckAndSet(ctx context.Context, id string) (*Lease, error) {
	after := options.After
	upsert := true
	opts := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}
	res := b.coll.FindOneAndUpdate(ctx, bson.M{"_id": _id, "ds": ""}, bson.M{}, &opts)
	if res.Err() != nil {
		return nil, errors.Wrap(res.Err(), "error finding and updating mongo db document")
	}
	lease := Lease{}
	if err := res.Decode(&lease); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling returned lease mongo document")
	}
	return &lease, nil
}
