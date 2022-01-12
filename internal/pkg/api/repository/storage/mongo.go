package storage

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetMongoDataBase(dsn, name string) (db *mongo.Database, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(dsn).
		SetRegistry(
			bson.NewRegistryBuilder().
				RegisterEncoder(testkube.TestStep, testStepCodec{}).
				Build(),
		)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	return client.Database(name), nil
}
