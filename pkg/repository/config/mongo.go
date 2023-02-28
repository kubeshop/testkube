package config

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

const CollectionName = "config"
const Id = "api"

func NewMongoRepository(db *mongo.Database, opts ...Opt) *MongoRepository {
	r := &MongoRepository{
		Coll: db.Collection(CollectionName),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

type Opt func(*MongoRepository)

func WithMongoRepositoryCollection(collection *mongo.Collection) Opt {
	return func(r *MongoRepository) {
		r.Coll = collection
	}
}

type MongoRepository struct {
	Coll *mongo.Collection
}

func (r *MongoRepository) GetUniqueClusterId(ctx context.Context) (clusterId string, err error) {
	config := testkube.Config{}
	_ = r.Coll.FindOne(ctx, bson.M{"id": Id}).Decode(&config)

	// generate new cluster id and save if there is not already
	if config.ClusterId == "" {
		config.ClusterId = fmt.Sprintf("cluster%s", telemetry.GetMachineID())
		_, err := r.Upsert(ctx, config)
		return config.ClusterId, err
	}

	return config.ClusterId, nil
}

func (r *MongoRepository) GetTelemetryEnabled(ctx context.Context) (ok bool, err error) {
	config := testkube.Config{}
	err = r.Coll.FindOne(ctx, bson.M{"id": Id}).Decode(&config)
	return config.EnableTelemetry, err
}

func (r *MongoRepository) Get(ctx context.Context) (result testkube.Config, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"id": Id}).Decode(&result)
	return
}

func (r *MongoRepository) Upsert(ctx context.Context, result testkube.Config) (updated testkube.Config, err error) {
	upsert := true
	result.Id = Id
	_, err = r.Coll.ReplaceOne(ctx, bson.M{"id": Id}, result, &options.ReplaceOptions{Upsert: &upsert})
	return result, err
}
