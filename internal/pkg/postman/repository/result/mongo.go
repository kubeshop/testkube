package result

import (
	"context"

	"github.com/kubeshop/kubetest/pkg/api/executor"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const CollectionName = "execution-results"

func NewMongoRespository(db mongo.Database) *MongoRepository {
	return &MongoRepository{
		Coll: db.Collection(CollectionName),
	}
}

type MongoRepository struct {
	Coll *mongo.Collection
}

func (r *MongoRepository) GetBy(ctx context.Context, id string) (result executor.ExecutionResult, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	return
}

func (r *MongoRepository) Insert(ctx context.Context, result executor.ExecutionResult) (err error) {
	_, err = r.Coll.InsertOne(ctx, result)
	return
}
