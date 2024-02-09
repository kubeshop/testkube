package testresult

import (
	"context"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/repository/common"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var _ Repository = (*MongoRepository)(nil)

const CollectionName = "testresults"

func NewMongoRepository(db *mongo.Database, allowDiskUse, isDocDb bool, opts ...MongoRepositoryOpt) *MongoRepository {
	r := &MongoRepository{
		db:           db,
		Coll:         db.Collection(CollectionName),
		allowDiskUse: allowDiskUse,
		isDocDb:      isDocDb,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

type MongoRepository struct {
	db           *mongo.Database
	Coll         *mongo.Collection
	allowDiskUse bool
	isDocDb      bool
}

func WithMongoRepositoryCollection(collection *mongo.Collection) MongoRepositoryOpt {
	return func(r *MongoRepository) {
		r.Coll = collection
	}
}

type MongoRepositoryOpt func(*MongoRepository)

func (r *MongoRepository) Get(ctx context.Context, id string) (result testkube.TestSuiteExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&result)
	return *result.UnscapeDots(), err
}

func (r *MongoRepository) GetByNameAndTestSuite(ctx context.Context, name, testSuiteName string) (result testkube.TestSuiteExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"name": name, "testsuite.name": testSuiteName}).Decode(&result)
	return *result.UnscapeDots(), err
}

func (r *MongoRepository) slowGetLatestByTestSuite(ctx context.Context, testSuiteName string) (*testkube.TestSuiteExecution, error) {
	opts := options.Aggregate()
	pipeline := []bson.M{
		{"$project": bson.M{"testsuite.name": 1, "starttime": 1, "endtime": 1}},
		{"$match": bson.M{"testsuite.name": testSuiteName}},

		{"$addFields": bson.M{
			"updatetime": bson.M{"$max": bson.A{"$starttime", "$endtime"}},
		}},
		{"$group": bson.D{
			{Key: "_id", Value: "$testsuite.name"},
			{Key: "doc", Value: bson.M{"$max": bson.D{
				{Key: "updatetime", Value: "$updatetime"},
				{Key: "content", Value: "$$ROOT"},
			}}},
		}},
		{"$sort": bson.M{"doc.updatetime": -1}},
		{"$limit": 1},

		{"$lookup": bson.M{"from": r.Coll.Name(), "localField": "doc.content._id", "foreignField": "_id", "as": "execution"}},
		{"$replaceRoot": bson.M{"newRoot": bson.M{"$arrayElemAt": bson.A{"$execution", 0}}}},
	}
	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	var items []testkube.TestSuiteExecution
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return items[0].UnscapeDots(), err
}

func (r *MongoRepository) GetLatestByTestSuite(ctx context.Context, testSuiteName string) (*testkube.TestSuiteExecution, error) {
	// Workaround, to use subset of MongoDB features available in AWS DocumentDB
	if r.isDocDb {
		return r.slowGetLatestByTestSuite(ctx, testSuiteName)
	}

	opts := options.Aggregate()
	pipeline := []bson.M{
		{"$group": bson.M{"_id": "$testsuite.name", "doc": bson.M{"$first": bson.M{}}}},
		{"$project": bson.M{"_id": 0, "name": "$_id"}},
		{"$match": bson.M{"name": testSuiteName}},

		{"$lookup": bson.M{"from": r.Coll.Name(), "let": bson.M{"name": "$name"}, "pipeline": []bson.M{
			{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$testsuite.name", "$$name"}}}},
			{"$sort": bson.M{"starttime": -1}},
			{"$limit": 1},
		}, "as": "execution_by_start_time"}},
		{"$lookup": bson.M{"from": r.Coll.Name(), "let": bson.M{"name": "$name"}, "pipeline": []bson.M{
			{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$testsuite.name", "$$name"}}}},
			{"$sort": bson.M{"endtime": -1}},
			{"$limit": 1},
		}, "as": "execution_by_end_time"}},
		{"$project": bson.M{"executions": bson.M{"$concatArrays": bson.A{"$execution_by_start_time", "$execution_by_end_time"}}}},
		{"$unwind": "$executions"},
		{"$replaceRoot": bson.M{"newRoot": "$executions"}},

		{"$group": bson.D{
			{Key: "_id", Value: "$testsuite.name"},
			{Key: "doc", Value: bson.M{"$max": bson.D{
				{Key: "updatetime", Value: bson.M{"$max": bson.A{"$starttime", "$endtime"}}},
				{Key: "content", Value: "$$ROOT"},
			}}},
		}},
		{"$sort": bson.M{"doc.updatetime": -1}},
		{"$replaceRoot": bson.M{"newRoot": "$doc.content"}},
		{"$limit": 1},
	}
	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	var items []testkube.TestSuiteExecution
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return items[0].UnscapeDots(), err
}

func (r *MongoRepository) slowGetLatestByTestSuites(ctx context.Context, testSuiteNames []string) (executions []testkube.TestSuiteExecution, err error) {
	documents := bson.A{}
	for _, testSuiteName := range testSuiteNames {
		documents = append(documents, bson.M{"testsuite.name": testSuiteName})
	}

	pipeline := []bson.M{
		{"$project": bson.M{"testsuite.name": 1, "starttime": 1, "endtime": 1}},
		{"$match": bson.M{"$or": documents}},

		{"$addFields": bson.M{
			"updatetime": bson.M{"$max": bson.A{"$starttime", "$endtime"}},
		}},
		{"$group": bson.D{
			{Key: "_id", Value: "$testsuite.name"},
			{Key: "doc", Value: bson.M{"$max": bson.D{
				{Key: "updatetime", Value: "$updatetime"},
				{Key: "content", Value: "$$ROOT"},
			}}},
		}},
		{"$sort": bson.M{"doc.updatetime": -1}},

		{"$lookup": bson.M{"from": r.Coll.Name(), "localField": "doc.content._id", "foreignField": "_id", "as": "execution"}},
		{"$replaceRoot": bson.M{"newRoot": bson.M{"$arrayElemAt": bson.A{"$execution", 0}}}},
	}

	opts := options.Aggregate()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &executions)
	if err != nil {
		return nil, err
	}

	if len(executions) == 0 {
		return executions, nil
	}

	for i := range executions {
		executions[i].UnscapeDots()
	}
	return executions, nil
}

func (r *MongoRepository) GetLatestByTestSuites(ctx context.Context, testSuiteNames []string) (executions []testkube.TestSuiteExecution, err error) {
	if len(testSuiteNames) == 0 {
		return executions, nil
	}

	// Workaround, to use subset of MongoDB features available in AWS DocumentDB
	if r.isDocDb {
		return r.slowGetLatestByTestSuites(ctx, testSuiteNames)
	}

	documents := bson.A{}
	for _, testSuiteName := range testSuiteNames {
		documents = append(documents, bson.M{"name": testSuiteName})
	}

	pipeline := []bson.M{
		{"$group": bson.M{"_id": "$testsuite.name", "doc": bson.M{"$first": bson.M{}}}},
		{"$project": bson.M{"_id": 0, "name": "$_id"}},
		{"$match": bson.M{"$or": documents}},

		{"$lookup": bson.M{"from": r.Coll.Name(), "let": bson.M{"name": "$name"}, "pipeline": []bson.M{
			{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$testsuite.name", "$$name"}}}},
			{"$sort": bson.M{"starttime": -1}},
			{"$limit": 1},
		}, "as": "execution_by_start_time"}},
		{"$lookup": bson.M{"from": r.Coll.Name(), "let": bson.M{"name": "$name"}, "pipeline": []bson.M{
			{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$testsuite.name", "$$name"}}}},
			{"$sort": bson.M{"endtime": -1}},
			{"$limit": 1},
		}, "as": "execution_by_end_time"}},
		{"$project": bson.M{"executions": bson.M{"$concatArrays": bson.A{"$execution_by_start_time", "$execution_by_end_time"}}}},
		{"$unwind": "$executions"},
		{"$replaceRoot": bson.M{"newRoot": "$executions"}},

		{"$group": bson.D{
			{Key: "_id", Value: "$testsuite.name"},
			{Key: "doc", Value: bson.M{"$max": bson.D{
				{Key: "updatetime", Value: bson.M{"$max": bson.A{"$starttime", "$endtime"}}},
				{Key: "content", Value: "$$ROOT"},
			}}},
		}},
		{"$sort": bson.M{"doc.updatetime": -1}},
		{"$replaceRoot": bson.M{"newRoot": "$doc.content"}},
	}

	opts := options.Aggregate()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &executions)
	if err != nil {
		return nil, err
	}

	if len(executions) == 0 {
		return executions, nil
	}

	for i := range executions {
		executions[i].UnscapeDots()
	}
	return executions, nil
}

func (r *MongoRepository) GetNewestExecutions(ctx context.Context, limit int) (result []testkube.TestSuiteExecution, err error) {
	result = make([]testkube.TestSuiteExecution, 0)
	resultLimit := int64(limit)
	opts := &options.FindOptions{Limit: &resultLimit}
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return result, err
	}
	err = cursor.All(ctx, &result)

	for i := range result {
		result[i].UnscapeDots()
	}
	return
}

func (r *MongoRepository) Count(ctx context.Context, filter Filter) (count int64, err error) {
	query, _ := composeQueryAndOpts(filter)
	return r.Coll.CountDocuments(ctx, query)
}

func (r *MongoRepository) GetExecutionsTotals(ctx context.Context, filter ...Filter) (totals testkube.ExecutionsTotals, err error) {
	var result []struct {
		Status string `bson:"_id"`
		Count  int    `bson:"count"`
	}

	query := bson.M{}
	if len(filter) > 0 {
		query, _ = composeQueryAndOpts(filter[0])
	}

	pipeline := []bson.D{
		{{Key: "$sort", Value: bson.M{"status": 1}}},
		{{Key: "$match", Value: query}},
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "starttime", Value: -1}}}})
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: int64(filter[0].Page() * filter[0].PageSize())}})
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: int64(filter[0].PageSize())}})
	} else {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "status", Value: 1}}}})
	}

	pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$status"},
		{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}})

	opts := options.Aggregate()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return totals, err
	}
	err = cursor.All(ctx, &result)
	if err != nil {
		return totals, err
	}

	var sum int32

	for _, o := range result {
		sum += int32(o.Count)
		switch testkube.TestSuiteExecutionStatus(o.Status) {
		case testkube.QUEUED_TestSuiteExecutionStatus:
			totals.Queued = int32(o.Count)
		case testkube.RUNNING_TestSuiteExecutionStatus:
			totals.Running = int32(o.Count)
		case testkube.PASSED_TestSuiteExecutionStatus:
			totals.Passed = int32(o.Count)
		case testkube.FAILED_TestSuiteExecutionStatus:
			totals.Failed = int32(o.Count)
		}
	}
	totals.Results = sum

	return
}

func (r *MongoRepository) GetExecutions(ctx context.Context, filter Filter) (result []testkube.TestSuiteExecution, err error) {
	result = make([]testkube.TestSuiteExecution, 0)
	query, opts := composeQueryAndOpts(filter)
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Find(ctx, query, opts)
	if err != nil {
		return
	}
	err = cursor.All(ctx, &result)

	for i := range result {
		result[i].UnscapeDots()
	}
	return
}

func (r *MongoRepository) Insert(ctx context.Context, result testkube.TestSuiteExecution) (err error) {
	result.EscapeDots()
	result.CleanStepsOutput()
	_, err = r.Coll.InsertOne(ctx, result)
	return
}

func (r *MongoRepository) Update(ctx context.Context, result testkube.TestSuiteExecution) (err error) {
	result.EscapeDots()
	result.CleanStepsOutput()
	_, err = r.Coll.ReplaceOne(ctx, bson.M{"id": result.Id}, result)
	return
}

// StartExecution updates execution start time
func (r *MongoRepository) StartExecution(ctx context.Context, id string, startTime time.Time) (err error) {
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"starttime": startTime}})
	return
}

// EndExecution updates execution end time
func (r *MongoRepository) EndExecution(ctx context.Context, e testkube.TestSuiteExecution) (err error) {
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": e.Id}, bson.M{"$set": bson.M{"endtime": e.EndTime, "duration": e.Duration, "durationms": e.DurationMs}})
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

	if filter.LastNDaysDefined() {
		startTimeQuery["$gte"] = time.Now().Add(-time.Duration(filter.LastNDays()) * 24 * time.Hour)
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

// DeleteByTestSuite deletes execution results by test suite
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

	var filter bson.M
	if len(testSuiteNames) > 1 {
		conditions := bson.A{}
		for _, testSuiteName := range testSuiteNames {
			conditions = append(conditions, bson.M{"testsuite.name": testSuiteName})
		}

		filter = bson.M{"$or": conditions}
	} else {
		filter = bson.M{"testsuite.name": testSuiteNames[0]}
	}

	_, err = r.Coll.DeleteMany(ctx, filter)
	return
}

// GetTestSuiteMetrics returns test executions metrics
func (r *MongoRepository) GetTestSuiteMetrics(ctx context.Context, name string, limit, last int) (metrics testkube.ExecutionsMetrics, err error) {
	query := bson.M{"testsuite.name": name}

	var pipeline []bson.D
	if last > 0 {
		query["starttime"] = bson.M{"$gte": time.Now().Add(-time.Duration(last) * 24 * time.Hour)}
	}

	pipeline = append(pipeline, bson.D{{Key: "$match", Value: query}})
	pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "starttime", Value: -1}}}})
	pipeline = append(pipeline, bson.D{
		{
			Key: "$project", Value: bson.D{
				{Key: "status", Value: 1},
				{Key: "duration", Value: 1},
				{Key: "starttime", Value: 1},
				{Key: "name", Value: 1},
			},
		},
	})

	opts := options.Aggregate()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return metrics, err
	}

	var executions []testkube.ExecutionsMetricsExecutions
	err = cursor.All(ctx, &executions)

	if err != nil {
		return metrics, err
	}

	metrics = common.CalculateMetrics(executions)
	if limit > 0 && limit < len(metrics.Executions) {
		metrics.Executions = metrics.Executions[:limit]
	}

	return metrics, nil
}
