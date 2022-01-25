package result

import (
	"context"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
	result = make([]testkube.Execution, 0)
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
	result = make([]testkube.Execution, 0)
	query, opts := composeQueryAndOpts(filter)
	cursor, err := r.Coll.Find(ctx, query, opts)
	if err != nil {
		return
	}
	err = cursor.All(ctx, &result)
	return
}

func (r *MongoRepository) GetExecutionTotals(ctx context.Context, filter ...Filter) (totals testkube.ExecutionsTotals, err error) {
	var result []struct {
		Status string `bson:"_id"`
		Count  int32  `bson:"count"`
	}

	query := bson.M{}
	if len(filter) > 0 {
		query, _ = composeQueryAndOpts(filter[0])
	}

	pipeline := []bson.D{{{"$match", query}}}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{"$sort", bson.D{{"starttime", -1}}}})
		pipeline = append(pipeline, bson.D{{"$skip", int64(filter[0].Page() * filter[0].PageSize())}})
		pipeline = append(pipeline, bson.D{{"$limit", int64(filter[0].PageSize())}})
	}

	pipeline = append(pipeline, bson.D{{"$group", bson.D{{"_id", "$executionresult.status"}, {"count", bson.D{{"$sum", 1}}}}}})
	cursor, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return totals, err
	}
	err = cursor.All(ctx, &result)
	if err != nil {
		return totals, err
	}

	var sum int32

	// TODO: statuses are messy e.g. success==passed error==failed
	for _, o := range result {
		sum += o.Count
		switch testkube.TestStatus(o.Status) {
		case testkube.QUEUED_TestStatus:
			totals.Queued = o.Count
		case testkube.PENDING_TestStatus:
			totals.Pending = o.Count
		case testkube.SUCCESS_TestStatus:
			totals.Passed = o.Count
		case testkube.ERROR__TestStatus:
			totals.Failed = o.Count
		}
	}
	totals.Results = sum

	return
}

func (r *MongoRepository) GetTags(ctx context.Context) (tags []string, err error) {
	var result []struct {
		Id   string   `bson:"_id"`
		Tags []string `bson:"tags"`
	}

	cursor, err := r.Coll.Aggregate(ctx, mongo.Pipeline{
		bson.D{{"$unwind", "$tags"}},
		bson.D{{"$group", bson.D{{"_id", "alltags"}, {"tags", bson.D{{"$addToSet", "$tags"}}}}}},
	})
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &result)
	if err != nil {
		return nil, err
	}
	tags = []string{}
	if len(result) > 0 {
		tags = result[0].Tags
		sort.Sort(sort.StringSlice(tags))
	}
	return tags, nil
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
func (r *MongoRepository) EndExecution(ctx context.Context, id string, endTime time.Time, duration time.Duration) (err error) {
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"endtime": endTime, "duration": duration.String()}})
	return
}

func composeQueryAndOpts(filter Filter) (bson.M, *options.FindOptions) {

	query := bson.M{}
	opts := options.Find()
	startTimeQuery := bson.M{}

	if filter.TextSearchDefined() {
		query["$or"] = bson.A{
			bson.M{"scriptname": bson.M{"$regex": primitive.Regex{Pattern: filter.TextSearch(), Options: "i"}}},
			bson.M{"name": bson.M{"$regex": primitive.Regex{Pattern: filter.TextSearch(), Options: "i"}}},
		}
	}

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
		query["starttime"] = startTimeQuery
	}

	if filter.StatusDefined() {
		query["executionresult.status"] = filter.Status()
	}

	if filter.Tags() != nil {
		query["tags"] = filter.Tags()
	}

	opts.SetSkip(int64(filter.Page() * filter.PageSize()))
	opts.SetLimit(int64(filter.PageSize()))
	opts.SetSort(bson.D{{"starttime", -1}})

	return query, opts
}
