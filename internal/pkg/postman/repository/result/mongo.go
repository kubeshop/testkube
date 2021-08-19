package result

import (
	"context"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const CollectionName = "executions"

func NewMongoRespository(db *mongo.Database) *MongoRepository {
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
	err = r.Coll.FindOneAndUpdate(ctx, bson.M{"status": kubtest.ExecutionStatusQueued}, bson.M{"$set": bson.M{"status": kubtest.ExecutionStatusPending}}).Decode(&result)
	return
}
