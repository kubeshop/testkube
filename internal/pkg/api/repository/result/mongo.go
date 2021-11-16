package result

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const CollectionName = "results"

func NewMongoRespository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		Coll: db.Collection(CollectionName),
	}
}

type MongoRepository struct {
	Coll *mongo.Collection
}

func (r *MongoRepository) Get(ctx context.Context, id string) (result testkube.Execution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	return
}

func (r *MongoRepository) GetByNameAndScript(ctx context.Context, name, script string) (result testkube.Execution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"name": name, "scriptname": script}).Decode(&result)
	return
}

func (r *MongoRepository) GetNewestExecutions(ctx context.Context, limit int) (result []testkube.Execution, err error) {
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

func (r *MongoRepository) GetExecutions(ctx context.Context, filter Filter) (result []testkube.Execution, err error) {
	query, opts := composeQueryAndOpts(filter)
	cursor, err := r.Coll.Find(ctx, query, opts)

	if err != nil {
		return
	}
	err = cursor.All(ctx, &result)
	return
}

func (r *MongoRepository) GetExecutionTotals(ctx context.Context, filter Filter) (result testkube.ExecutionsTotals, err error) {

	query, _ := composeQueryAndOpts(filter)
	query["scriptType"] = bson.M{"$exists": true}
	total, err := r.Coll.CountDocuments(ctx, query)
	if err != nil {
		return
	}
	result.Results = int32(total)

	query["executionResult"] = bson.M{"status": testkube.QUEUED_ExecutionStatus}
	queued, err := r.Coll.CountDocuments(ctx, query)
	if err != nil {
		return
	}
	result.Queued = int32(queued)

	query["executionResult"] = bson.M{"status": testkube.PENDING_ExecutionStatus}
	pending, err := r.Coll.CountDocuments(ctx, query)
	if err != nil {
		return
	}
	result.Pending = int32(pending)

	query["executionResult"] = bson.M{"status": testkube.ERROR__ExecutionStatus}
	failed, err := r.Coll.CountDocuments(ctx, query)
	if err != nil {
		return
	}
	result.Failed = int32(failed)
	return
}

func (r *MongoRepository) Insert(ctx context.Context, result testkube.Execution) (err error) {
	_, err = r.Coll.InsertOne(ctx, result)
	return
}

func (r *MongoRepository) Update(ctx context.Context, result testkube.Execution) (err error) {
	_, err = r.Coll.ReplaceOne(ctx, bson.M{"id": result.Id}, result)
	return
}

func (r *MongoRepository) UpdateResult(ctx context.Context, id string, result testkube.ExecutionResult) (err error) {
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"executionresult": result}})
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

	if filter.ScriptNameDefined() {
		query["scriptname"] = filter.ScriptName()
	}

	if filter.StartDateDefined() {
		startTimeQuery["$gte"] = filter.StartDate()
	}

	if filter.EndDateDefined() {
		startTimeQuery["$lte"] = filter.EndDate()
	}

	if len(startTimeQuery) > 0 {
		query["startTime"] = startTimeQuery
	}

	if filter.StatusDefined() {
		query["executionResult"] = bson.M{"status": filter.Status()}
	}

	opts.SetSkip(int64(filter.Page() * filter.PageSize()))
	opts.SetLimit(int64(filter.PageSize()))

	return query, opts
}
