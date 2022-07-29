package result

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	CollectionName    = "results"
	CollectionNumbers = "numbers"
)

func NewMongoRespository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		Coll:    db.Collection(CollectionName),
		Numbers: db.Collection(CollectionNumbers),
	}
}

type MongoRepository struct {
	Coll    *mongo.Collection
	Numbers *mongo.Collection
}

type executionNumber struct {
	TestName string `json:"testName"`
	Number   int    `json:"number"`
}

func (r *MongoRepository) Get(ctx context.Context, id string) (result testkube.Execution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	return
}
func (r *MongoRepository) GetByName(ctx context.Context, name string) (result testkube.Execution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"name": name}).Decode(&result)
	return
}

func (r *MongoRepository) GetByNameAndTest(ctx context.Context, name, testName string) (result testkube.Execution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"name": name, "testname": testName}).Decode(&result)
	return
}

func (r *MongoRepository) GetLatestByTest(ctx context.Context, testName, sortField string) (result testkube.Execution, err error) {
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{Key: sortField, Value: -1}})
	err = r.Coll.FindOne(ctx, bson.M{"testname": testName}, findOptions).Decode(&result)
	return
}

func (r *MongoRepository) GetLatestByTests(ctx context.Context, testNames []string, sortField string) (executions []testkube.Execution, err error) {
	var results []struct {
		LatestID string `bson:"latest_id"`
	}

	if len(testNames) == 0 {
		return executions, nil
	}

	conditions := bson.A{}
	for _, testName := range testNames {
		conditions = append(conditions, bson.M{"testname": testName})
	}

	pipeline := []bson.D{{{Key: "$match", Value: bson.M{"$or": conditions}}}}
	pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: sortField, Value: -1}}}})
	pipeline = append(pipeline, bson.D{
		{Key: "$group", Value: bson.D{{Key: "_id", Value: "$testname"}, {Key: "latest_id", Value: bson.D{{Key: "$first", Value: "$id"}}}}}})

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

func (r *MongoRepository) GetNewestExecutions(ctx context.Context, limit int) (result []testkube.Execution, err error) {
	result = make([]testkube.Execution, 0)
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

func (r *MongoRepository) GetExecutionTotals(ctx context.Context, paging bool, filter ...Filter) (totals testkube.ExecutionsTotals, err error) {
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
		if paging {
			pipeline = append(pipeline, bson.D{{Key: "$skip", Value: int64(filter[0].Page() * filter[0].PageSize())}})
			pipeline = append(pipeline, bson.D{{Key: "$limit", Value: int64(filter[0].PageSize())}})
		}
	}

	pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$executionresult.status"},
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

func (r *MongoRepository) GetLabels(ctx context.Context) (labels map[string][]string, err error) {
	var result []struct {
		Labels bson.M `bson:"labels"`
	}

	cursor, err := r.Coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	err = cursor.All(ctx, &result)
	if err != nil {
		return nil, err
	}

	labels = map[string][]string{}
	for _, r := range result {
		for key, value := range r.Labels {
			if values, ok := labels[key]; !ok {
				labels[key] = []string{fmt.Sprint(value)}
			} else {
				for _, v := range values {
					if v == value {
						continue
					}
				}
				labels[key] = append(labels[key], fmt.Sprint(value))
			}
		}
	}
	return labels, nil
}

func (r *MongoRepository) GetNextExecutionNumber(ctx context.Context, testName string) (number int, err error) {

	execNmbr := executionNumber{}
	retry := false
	retryAttempts := 0
	maxRetries := 10

	opts := options.FindOneAndUpdate()
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)

	err = r.Numbers.FindOne(context.Background(), bson.M{"testname": testName}).Decode(&execNmbr)
	if err == mongo.ErrNoDocuments {
		execution, err := r.GetLatestByTest(context.Background(), testName, "number")
		if err == mongo.ErrNoDocuments || execution.Number == 0 || err != nil {
			execNmbr.TestName = testName
			execNmbr.Number = 1
			_, err = r.Numbers.InsertOne(ctx, execNmbr)
			retry = (err != nil)
		}
	} else {
		err = r.Numbers.FindOneAndUpdate(ctx, bson.M{"testname": testName}, bson.M{"$inc": bson.M{"number": 1}}, opts).Decode(&execNmbr)
		retry = (err != nil)
	}

	for retry {
		retryAttempts++
		err = r.Numbers.FindOneAndUpdate(ctx, bson.M{"testname": testName}, bson.M{"$inc": bson.M{"number": 1}}, opts).Decode(&execNmbr)
		if err == nil || retryAttempts >= maxRetries {
			retry = false
		}
	}

	return execNmbr.Number, nil
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
	conditions := bson.A{}
	opts := options.Find()
	startTimeQuery := bson.M{}

	if filter.TextSearchDefined() {
		conditions = append(conditions, bson.M{"$or": bson.A{
			bson.M{"testname": bson.M{"$regex": primitive.Regex{Pattern: filter.TextSearch(), Options: "i"}}},
			bson.M{"name": bson.M{"$regex": primitive.Regex{Pattern: filter.TextSearch(), Options: "i"}}},
		}})
	}

	if filter.TestNameDefined() {
		conditions = append(conditions, bson.M{"testname": filter.TestName()})
	}

	if filter.StartDateDefined() {
		startTimeQuery["$gte"] = filter.StartDate()
	}

	if filter.EndDateDefined() {
		startTimeQuery["$lte"] = filter.EndDate()
	}

	if len(startTimeQuery) > 0 {
		conditions = append(conditions, bson.M{"starttime": startTimeQuery})
	}

	if filter.StatusesDefined() {
		statuses := filter.Statuses()
		if len(statuses) == 1 {
			conditions = append(conditions, bson.M{"executionresult.status": statuses[0]})
		} else {
			expressions := bson.A{}
			for _, status := range statuses {
				expressions = append(expressions, bson.M{"executionresult.status": status})
			}

			conditions = append(conditions, bson.M{"$or": expressions})
		}
	}

	if filter.Selector() != "" {
		conditions = addSelectorConditions(filter.Selector(), "labels", conditions)
	}

	if filter.TypeDefined() {
		conditions = append(conditions, bson.M{"testtype": filter.Type()})
	}

	opts.SetSkip(int64(filter.Page() * filter.PageSize()))
	opts.SetLimit(int64(filter.PageSize()))
	opts.SetSort(bson.D{{Key: "starttime", Value: -1}})

	if len(conditions) > 0 {
		query = bson.M{"$and": conditions}
	}

	return query, opts
}

func addSelectorConditions(selector string, tag string, conditions primitive.A) primitive.A {
	items := strings.Split(selector, ",")
	for _, item := range items {
		elements := strings.Split(item, "=")
		if len(elements) == 2 {
			conditions = append(conditions, bson.M{tag + "." + elements[0]: elements[1]})
		} else if len(elements) == 1 {
			conditions = append(conditions, bson.M{tag + "." + elements[0]: bson.M{"$exists": true}})
		}
	}
	return conditions
}

// DeleteByTest deletes execution results by test
func (r *MongoRepository) DeleteByTest(ctx context.Context, testName string) (err error) {
	_, err = r.Coll.DeleteMany(ctx, bson.M{"testname": testName})
	return
}

// DeleteByTestSuite deletes execution results by test suite
func (r *MongoRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) (err error) {
	_, err = r.Coll.DeleteMany(ctx, bson.M{"testsuitename": testSuiteName})
	return
}

// DeleteAll deletes all execution results
func (r *MongoRepository) DeleteAll(ctx context.Context) (err error) {
	_, err = r.Coll.DeleteMany(ctx, bson.M{})
	return
}

// DeleteByTests deletes execution results by tests
func (r *MongoRepository) DeleteByTests(ctx context.Context, testNames []string) (err error) {
	if len(testNames) == 0 {
		return nil
	}

	var filter bson.M
	if len(testNames) > 1 {
		conditions := bson.A{}
		for _, testName := range testNames {
			conditions = append(conditions, bson.M{"testname": testName})
		}

		filter = bson.M{"$or": conditions}
	} else {
		filter = bson.M{"testname": testNames[0]}
	}

	_, err = r.Coll.DeleteMany(ctx, filter)
	return
}

// DeleteByTestSuites deletes execution results by test suites
func (r *MongoRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) (err error) {
	if len(testSuiteNames) == 0 {
		return nil
	}

	var filter bson.M
	if len(testSuiteNames) > 1 {
		conditions := bson.A{}
		for _, testSuiteName := range testSuiteNames {
			conditions = append(conditions, bson.M{"testsuitename": testSuiteName})
		}

		filter = bson.M{"$or": conditions}
	} else {
		filter = bson.M{"testSuitename": testSuiteNames[0]}
	}

	_, err = r.Coll.DeleteMany(ctx, filter)
	return
}

// DeleteForAllTestSuites deletes execution results for all test suites
func (r *MongoRepository) DeleteForAllTestSuites(ctx context.Context) (err error) {
	_, err = r.Coll.DeleteMany(ctx, bson.M{"testsuitename": bson.M{"$ne": ""}})
	return
}
