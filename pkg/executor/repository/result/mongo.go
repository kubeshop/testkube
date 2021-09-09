package result

import (
	"context"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const CollectionName = "executions"

// NewMongoRespository creates new result repository with db setup for given collection
// use empty collection name as param for default "executions" collection name
func NewMongoRespository(db *mongo.Database, collection string) *MongoRepository {
	if collection == "" {
		collection = CollectionName
	}

	return &MongoRepository{
		Coll: db.Collection(CollectionName),
	}
}

type MongoRepository struct {
	Coll *mongo.Collection
}

func (r *MongoRepository) Get(ctx context.Context, id string) (result kubtest.Execution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	return
}

func (r *MongoRepository) Insert(ctx context.Context, result kubtest.Execution) (err error) {
	_, err = r.Coll.InsertOne(ctx, result)
	return
}

func (r *MongoRepository) Update(ctx context.Context, result kubtest.Execution) (err error) {
	_, err = r.Coll.ReplaceOne(ctx, bson.M{"id": result.Id}, result)
	return
}

func (r *MongoRepository) QueuePull(ctx context.Context) (result kubtest.Execution, err error) {
	err = r.Coll.FindOneAndUpdate(ctx, bson.M{"status": kubtest.QUEUED_ExecutionStatus}, bson.M{"$set": bson.M{"status": kubtest.PENDING_ExecutionStatus}}).Decode(&result)
	return
}
