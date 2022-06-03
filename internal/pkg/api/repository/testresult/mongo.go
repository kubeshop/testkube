package testresult

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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

func (r *MongoRepository) Get(ctx context.Context, id string) (result testkube.TestSuiteExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	return
}

func (r *MongoRepository) GetByNameAndTestSuite(ctx context.Context, name, testSuiteName string) (result testkube.TestSuiteExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"name": name, "testsuite.name": testSuiteName}).Decode(&result)
	return
}

func (r *MongoRepository) GetLatestByTestSuite(ctx context.Context, testSuiteName, sortField string) (result testkube.TestSuiteExecution, err error) {
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{Key: sortField, Value: -1}})
	err = r.Coll.FindOne(ctx, bson.M{"testsuite.name": testSuiteName}, findOptions).Decode(&result)
	return
}

func (r *MongoRepository) GetLatestByTestSuites(ctx context.Context, testSuiteNames []string, sortField string) (executions []testkube.TestSuiteExecution, err error) {
	var results []struct {
		LatestID string `bson:"latest_id"`
	}

	if len(testSuiteNames) == 0 {
		return executions, nil
	}

	conditions := bson.A{}
	for _, testSuiteName := range testSuiteNames {
		conditions = append(conditions, bson.M{"testsuite.name": testSuiteName})
	}

	pipeline := []bson.D{{{Key: "$match", Value: bson.M{"$or": conditions}}}}
	pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: sortField, Value: -1}}}})
	pipeline = append(pipeline, bson.D{
		{Key: "$group", Value: bson.D{{Key: "_id", Value: "$testsuite.name"}, {Key: "latest_id", Value: bson.D{{Key: "$first", Value: "$id"}}}}}})

	cursor, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return executions, nil
	}

	conditions = bson.A{}
	for _, result := range results {
		conditions = append(conditions, bson.M{"id": result.LatestID})
	}

	cursor, err = r.Coll.Find(ctx, bson.M{"$or": conditions})
	if err != nil {
		return nil, err
	}

	err = cursor.All(ctx, &executions)
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func (r *MongoRepository) GetNewestExecutions(ctx context.Context, limit int) (result []testkube.TestSuiteExecution, err error) {
	result = make([]testkube.TestSuiteExecution, 0)
	resultLimit := int64(limit)
	opts := &options.FindOptions{Limit: &resultLimit}
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})
	cursor, err := r.Coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return result, err
	}
	err = cursor.All(ctx, &result)
	return
}

func (r *MongoRepository) GetExecutionsTotals(ctx context.Context, filter ...Filter) (totals testkube.ExecutionsTotals, err error) {
	var result []struct {
		Status string `bson:"_id"`
		Count  int32  `bson:"count"`
	}

	query := bson.M{}
	if len(filter) > 0 {
		query, _ = composeQueryAndOpts(filter[0])
	}

	pipeline := []bson.D{{{Key: "$match", Value: query}}}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "starttime", Value: -1}}}})
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: int64(filter[0].Page() * filter[0].PageSize())}})
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: int64(filter[0].PageSize())}})
	}

	pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$status"},
		{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}})
	cursor, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return totals, err
	}
	err = cursor.All(ctx, &result)
	if err != nil {
		return totals, err
	}

	var sum int32

	for _, o := range result {
		sum += o.Count
		switch testkube.TestSuiteExecutionStatus(o.Status) {
		case testkube.QUEUED_TestSuiteExecutionStatus:
			totals.Queued = o.Count
		case testkube.RUNNING_TestSuiteExecutionStatus:
			totals.Running = o.Count
		case testkube.PASSED_TestSuiteExecutionStatus:
			totals.Passed = o.Count
		case testkube.FAILED_TestSuiteExecutionStatus:
			totals.Failed = o.Count
		}
	}
	totals.Results = sum

	return
}

func (r *MongoRepository) GetExecutions(ctx context.Context, filter Filter) (result []testkube.TestSuiteExecution, err error) {
	result = make([]testkube.TestSuiteExecution, 0)
	query, opts := composeQueryAndOpts(filter)
	cursor, err := r.Coll.Find(ctx, query, opts)
	if err != nil {
		return
	}
	err = cursor.All(ctx, &result)
	return
}

func (r *MongoRepository) Insert(ctx context.Context, result testkube.TestSuiteExecution) (err error) {
	_, err = r.Coll.InsertOne(ctx, result)
	return
}

func (r *MongoRepository) Update(ctx context.Context, result testkube.TestSuiteExecution) (err error) {
	_, err = r.Coll.ReplaceOne(ctx, bson.M{"id": result.Id}, result)
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

	if filter.NameDefined() {
		query["testsuite.name"] = filter.Name()
	}

	if filter.TextSearchDefined() {
		query["name"] = bson.M{"$regex": primitive.Regex{Pattern: filter.TextSearch(), Options: "i"}}
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

	if filter.StatusesDefined() {
		statuses := filter.Statuses()
		if len(statuses) == 1 {
			query["status"] = statuses[0]
		} else {
			var conditions bson.A
			for _, status := range statuses {
				conditions = append(conditions, bson.M{"status": status})
			}

			query["$or"] = conditions
		}
	}

	if filter.Selector() != "" {
		items := strings.Split(filter.Selector(), ",")
		for _, item := range items {
			elements := strings.Split(item, "=")
			if len(elements) == 2 {
				query["labels."+elements[0]] = elements[1]
			} else if len(elements) == 1 {
				query["labels."+elements[0]] = bson.M{"$exists": true}
			}
		}
	}

	opts.SetSkip(int64(filter.Page() * filter.PageSize()))
	opts.SetLimit(int64(filter.PageSize()))
	opts.SetSort(bson.D{{Key: "starttime", Value: -1}})

	return query, opts
}

// DeleteByTest deletes execution results by test suite
func (r *MongoRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) (err error) {
	_, err = r.Coll.DeleteMany(ctx, bson.M{"testsuite.name": testSuiteName})
	return
}

// DeleteAll deletes all execution results
func (r *MongoRepository) DeleteAll(ctx context.Context) (err error) {
	_, err = r.Coll.DeleteMany(ctx, bson.M{})
	return
}

// DeleteByTestSuites deletes execution results by test suites
func (r *MongoRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) (err error) {
	if len(testSuiteNames) == 0 {
		return nil
	}

	conditions := bson.A{}
	for _, testSuiteName := range testSuiteNames {
		conditions = append(conditions, bson.M{"testsuite.name": testSuiteName})
	}

	filter := []bson.D{{{Key: "$match", Value: bson.M{"$or": conditions}}}}
	_, err = r.Coll.DeleteMany(ctx, filter)
	return
}
