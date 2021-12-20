package testresult

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const CollectionName = "testresults"

func NewMongoRespository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		Coll: db.Collection(CollectionName),
	}
}

type MongoRepository struct {
	Coll *mongo.Collection
}

func (r *MongoRepository) Get(ctx context.Context, id string) (result testkube.TestExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	return
}

func (r *MongoRepository) GetByNameAndScript(ctx context.Context, name, script string) (result testkube.TestExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"name": name, "scriptname": script}).Decode(&result)
	return
}

func (r *MongoRepository) GetNewestExecutions(ctx context.Context, limit int) (result []testkube.TestExecution, err error) {
	resultLimit := int64(limit)
	opts := &options.FindOptions{Limit: &resultLimit}
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})
	cursor, err := r.Coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return result, err
	}
	cursor.All(ctx, &result)
	return
}

func (r *MongoRepository) GetExecutions(ctx context.Context, filter Filter) (result []testkube.TestExecution, err error) {
	query, opts := composeQueryAndOpts(filter)
	cursor, err := r.Coll.Find(ctx, query, opts)
	if err != nil {
		return
	}
	err = cursor.All(ctx, &result)
	return
}

func (r *MongoRepository) Insert(ctx context.Context, result testkube.TestExecution) (err error) {
	_, err = r.Coll.InsertOne(ctx, result)
	return
}

func (r *MongoRepository) Update(ctx context.Context, result testkube.TestExecution) (err error) {
	_, err = r.Coll.ReplaceOne(ctx, bson.M{"id": result.Id}, result)
	return
}

// StartExecution updates execution start time
func (r *MongoRepository) StartExecution(ctx context.Context, id string, startTime time.Time) (err error) {
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"starttime": startTime}})
	return
}

// EndExecution updates execution end time
func (r *MongoRepository) EndExecution(ctx context.Context, id string, endTime time.Time) (err error) {
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"endtime": endTime}})
	return
}

func composeQueryAndOpts(filter Filter) (bson.M, *options.FindOptions) {

	query := bson.M{}
	opts := options.Find()
	startTimeQuery := bson.M{}

	if filter.TextSearchDefined() {
		query["$or"] = bson.A{
			bson.M{"scriptname": filter.TextSearch()},
			bson.M{"name": filter.TextSearch()},
		}
	}

	if filter.StartDateDefined() {
		startTimeQuery["$gte"] = filter.StartDate()
	}

	if filter.EndDateDefined() {
		startTimeQuery["$lte"] = filter.EndDate()
	}

	if len(startTimeQuery) > 0 {
		query["starttime"] = startTimeQuery
	}

	opts.SetSkip(int64(filter.Page() * filter.PageSize()))
	opts.SetLimit(int64(filter.PageSize()))
	opts.SetSort(bson.D{{"starttime", -1}})

	return query, opts
}
