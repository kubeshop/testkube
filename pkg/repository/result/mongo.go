package result

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/repository/common"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/log"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/storage"
)

var _ Repository = (*MongoRepository)(nil)

const (
	CollectionResults   = "results"
	CollectionSequences = "sequences"
	// OutputPrefixSize is the size of the beginning of execution output in case this doesn't fit into Mongo
	OutputPrefixSize = 1 * 1024 * 1024
	// OutputMaxSize is the size of the execution output in case this doesn't fit into the 16 MB limited by Mongo
	OutputMaxSize = 14 * 1024 * 1024
	// OverflownOutputWarn is the message that lets the user know the output had to be trimmed
	OverflownOutputWarn = "WARNING: Output was shortened in order to fit into MongoDB."
	// StepMaxCount is the maximum number of steps saved into Mongo - due to the 16 MB document size limitation
	StepMaxCount = 100
)

// NewMongoRepository creates a new MongoRepository - used by other testkube components - use opts to extend the functionality
func NewMongoRepository(db *mongo.Database, allowDiskUse, isDocDb bool, opts ...MongoRepositoryOpt) *MongoRepository {
	r := &MongoRepository{
		db:               db,
		ResultsColl:      db.Collection(CollectionResults),
		SequencesColl:    db.Collection(CollectionSequences),
		OutputRepository: NewMongoOutputRepository(db),
		allowDiskUse:     allowDiskUse,
		isDocDb:          isDocDb,
		log:              log.DefaultLogger,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func NewMongoRepositoryWithOutputRepository(
	db *mongo.Database,
	allowDiskUse bool,
	outputRepository OutputRepository,
	opts ...MongoRepositoryOpt,
) *MongoRepository {
	r := &MongoRepository{
		db:               db,
		ResultsColl:      db.Collection(CollectionResults),
		SequencesColl:    db.Collection(CollectionSequences),
		OutputRepository: outputRepository,
		allowDiskUse:     allowDiskUse,
		log:              log.DefaultLogger,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func NewMongoRepositoryWithMinioOutputStorage(db *mongo.Database, allowDiskUse bool, storageClient storage.Client, bucket string) *MongoRepository {
	repo := MongoRepository{
		db:            db,
		ResultsColl:   db.Collection(CollectionResults),
		SequencesColl: db.Collection(CollectionSequences),
		allowDiskUse:  allowDiskUse,
		log:           log.DefaultLogger,
	}
	repo.OutputRepository = NewMinioOutputRepository(storageClient, repo.ResultsColl, bucket)
	return &repo
}

type MongoRepository struct {
	db               *mongo.Database
	ResultsColl      *mongo.Collection
	SequencesColl    *mongo.Collection
	OutputRepository OutputRepository
	logGrpcClient    logsclient.StreamGetter
	allowDiskUse     bool
	isDocDb          bool
	features         featureflags.FeatureFlags
	log              *zap.SugaredLogger
}

type MongoRepositoryOpt func(*MongoRepository)

func WithLogsClient(client logsclient.StreamGetter) MongoRepositoryOpt {
	return func(r *MongoRepository) {
		r.logGrpcClient = client
	}
}

func WithFeatureFlags(features featureflags.FeatureFlags) MongoRepositoryOpt {
	return func(r *MongoRepository) {
		r.features = features
	}
}

func WithMongoRepositoryResultCollection(collection *mongo.Collection) MongoRepositoryOpt {
	return func(r *MongoRepository) {
		r.ResultsColl = collection
	}
}

func WithMongoRepositorySequenceCollection(collection *mongo.Collection) MongoRepositoryOpt {
	return func(r *MongoRepository) {
		r.SequencesColl = collection
	}
}

func (r *MongoRepository) getOutputFromLogServer(ctx context.Context, result *testkube.Execution) (string, error) {
	if r.logGrpcClient == nil {
		return "", nil
	}

	if result.ExecutionResult == nil || !result.ExecutionResult.IsCompleted() {
		return "", nil
	}

	logs, err := r.logGrpcClient.Get(ctx, result.Id)
	if err != nil {
		return "", err
	}

	output := ""
	for log := range logs {
		if log.Error != nil {
			r.log.Errorw("can't get log line", "error", log.Error)
			continue
		}

		output += log.Log.Content + "\n"
	}

	return output, nil
}

func (r *MongoRepository) GetExecution(ctx context.Context, id string) (result testkube.Execution, err error) {
	err = r.ResultsColl.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&result)
	if err != nil {
		return
	}
	return *result.UnscapeDots(), err
}

func (r *MongoRepository) Get(ctx context.Context, id string) (result testkube.Execution, err error) {
	err = r.ResultsColl.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&result)
	if err != nil {
		return
	}

	err = r.attachOutput(ctx, &result)
	if err != nil {
		return
	}
	return *result.UnscapeDots(), err
}

func (r *MongoRepository) GetByNameAndTest(ctx context.Context, name, testName string) (result testkube.Execution, err error) {
	err = r.ResultsColl.FindOne(ctx, bson.M{"name": name, "testname": testName}).Decode(&result)
	if err != nil {
		return
	}
	err = r.attachOutput(ctx, &result)
	if err != nil {
		return
	}

	return *result.UnscapeDots(), err
}

func (r *MongoRepository) attachOutput(ctx context.Context, result *testkube.Execution) (err error) {
	if len(result.ExecutionResult.Output) == 0 && !r.features.LogsV2 {
		result.ExecutionResult.Output, err = r.OutputRepository.GetOutput(ctx, result.Id, result.TestName, result.TestSuiteName)
		if err == mongo.ErrNoDocuments {
			err = nil
		}
	}
	return err
}

func (r *MongoRepository) GetLatestTestNumber(ctx context.Context, testName string) (num int32, err error) {
	var result struct {
		Number int32 `bson:"number"`
	}
	opts := &options.FindOneOptions{Projection: bson.M{"_id": -1, "number": 1}, Sort: bson.M{"number": -1}}
	err = r.ResultsColl.FindOne(ctx, bson.M{"testname": testName}, opts).Decode(&result)
	if err != nil {
		return 0, err
	}
	return result.Number, err
}

func (r *MongoRepository) slowGetLatestByTest(ctx context.Context, testName string) (*testkube.Execution, error) {
	opts := options.Aggregate()
	pipeline := []bson.M{
		{"$project": bson.M{"testname": 1, "starttime": 1, "endtime": 1}},
		{"$match": bson.M{"testname": testName}},

		{"$group": bson.D{
			{Key: "_id", Value: "$testname"},
			{Key: "doc", Value: bson.M{"$max": bson.D{
				{Key: "updatetime", Value: bson.M{"$max": bson.A{"$starttime", "$endtime"}}},
				{Key: "content", Value: "$$ROOT"},
			}}},
		}},
		{"$sort": bson.M{"doc.updatetime": -1}},
		{"$limit": 1},

		{"$lookup": bson.M{"from": r.ResultsColl.Name(), "localField": "doc.content._id", "foreignField": "_id", "as": "execution"}},
		{"$replaceRoot": bson.M{"newRoot": bson.M{"$arrayElemAt": bson.A{"$execution", 0}}}},
	}
	cursor, err := r.ResultsColl.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	var items []testkube.Execution
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	result := (&items[0]).UnscapeDots()

	err = r.attachOutput(ctx, result)
	if err != nil {
		return result, err
	}
	return result, err
}

func (r *MongoRepository) GetLatestByTest(ctx context.Context, testName string) (*testkube.Execution, error) {
	// Workaround, to use subset of MongoDB features available in AWS DocumentDB
	if r.isDocDb {
		return r.slowGetLatestByTest(ctx, testName)
	}

	opts := options.Aggregate()
	pipeline := []bson.M{
		{"$group": bson.M{"_id": "$testname", "doc": bson.M{"$first": bson.M{}}}},
		{"$project": bson.M{"_id": 0, "name": "$_id"}},
		{"$match": bson.M{"name": testName}},
		{"$lookup": bson.M{"from": r.ResultsColl.Name(), "let": bson.M{"name": "$name"}, "pipeline": []bson.M{
			{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$testname", "$$name"}}}},
			{"$sort": bson.M{"starttime": -1}},
			{"$limit": 1},
		}, "as": "execution_by_start_time"}},
		{"$lookup": bson.M{"from": r.ResultsColl.Name(), "let": bson.M{"name": "$name"}, "pipeline": []bson.M{
			{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$testname", "$$name"}}}},
			{"$sort": bson.M{"endtime": -1}},
			{"$limit": 1},
		}, "as": "execution_by_end_time"}},
		{"$project": bson.M{"executions": bson.M{"$concatArrays": bson.A{"$execution_by_start_time", "$execution_by_end_time"}}}},
		{"$unwind": "$executions"},
		{"$replaceRoot": bson.M{"newRoot": "$executions"}},

		{"$group": bson.D{
			{Key: "_id", Value: "$testname"},
			{Key: "doc", Value: bson.M{"$max": bson.D{
				{Key: "updatetime", Value: bson.M{"$max": bson.A{"$starttime", "$endtime"}}},
				{Key: "content", Value: "$$ROOT"},
			}}},
		}},
		{"$sort": bson.M{"doc.updatetime": -1}},
		{"$replaceRoot": bson.M{"newRoot": "$doc.content"}},
		{"$limit": 1},
	}
	cursor, err := r.ResultsColl.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	var items []testkube.Execution
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	result := (&items[0]).UnscapeDots()

	err = r.attachOutput(ctx, result)
	if err != nil {
		return result, err
	}

	return result, err
}

func (r *MongoRepository) slowGetLatestByTests(ctx context.Context, testNames []string) (executions []testkube.Execution, err error) {
	documents := bson.A{}
	for _, testName := range testNames {
		documents = append(documents, bson.M{"testname": testName})
	}

	pipeline := []bson.M{
		{"$project": bson.M{"testname": 1, "starttime": 1, "endtime": 1}},
		{"$match": bson.M{"$or": documents}},

		{"$group": bson.D{
			{Key: "_id", Value: "$testname"},
			{Key: "doc", Value: bson.M{"$max": bson.D{
				{Key: "updatetime", Value: bson.M{"$max": bson.A{"$starttime", "$endtime"}}},
				{Key: "content", Value: "$$ROOT"},
			}}},
		}},
		{"$sort": bson.M{"doc.updatetime": -1}},

		{"$lookup": bson.M{"from": r.ResultsColl.Name(), "localField": "doc.content._id", "foreignField": "_id", "as": "execution"}},
		{"$replaceRoot": bson.M{"newRoot": bson.M{"$arrayElemAt": bson.A{"$execution", 0}}}},
	}

	opts := options.Aggregate()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.ResultsColl.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &executions)
	if err != nil {
		return nil, err
	}

	for i := range executions {
		executions[i].UnscapeDots()
	}
	return executions, nil
}

func (r *MongoRepository) GetLatestByTests(ctx context.Context, testNames []string) (executions []testkube.Execution, err error) {
	if len(testNames) == 0 {
		return executions, nil
	}

	// Workaround, to use subset of MongoDB features available in AWS DocumentDB
	if r.isDocDb {
		return r.slowGetLatestByTests(ctx, testNames)
	}

	documents := bson.A{}
	for _, testName := range testNames {
		documents = append(documents, bson.M{"name": testName})
	}

	pipeline := []bson.M{
		{"$group": bson.M{"_id": "$testname", "doc": bson.M{"$first": bson.M{}}}},
		{"$project": bson.M{"_id": 0, "name": "$_id"}},
		{"$match": bson.M{"$or": documents}},

		{"$lookup": bson.M{"from": r.ResultsColl.Name(), "let": bson.M{"name": "$name"}, "pipeline": []bson.M{
			{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$testname", "$$name"}}}},
			{"$sort": bson.M{"starttime": -1}},
			{"$limit": 1},
		}, "as": "execution_by_start_time"}},
		{"$lookup": bson.M{"from": r.ResultsColl.Name(), "let": bson.M{"name": "$name"}, "pipeline": []bson.M{
			{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$testname", "$$name"}}}},
			{"$sort": bson.M{"endtime": -1}},
			{"$limit": 1},
		}, "as": "execution_by_end_time"}},
		{"$project": bson.M{"executions": bson.M{"$concatArrays": bson.A{"$execution_by_start_time", "$execution_by_end_time"}}}},
		{"$unwind": "$executions"},
		{"$replaceRoot": bson.M{"newRoot": "$executions"}},

		{"$group": bson.D{
			{Key: "_id", Value: "$testname"},
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

	cursor, err := r.ResultsColl.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &executions)
	if err != nil {
		return nil, err
	}

	for i := range executions {
		executions[i].UnscapeDots()
	}
	return executions, nil
}

func (r *MongoRepository) GetNewestExecutions(ctx context.Context, limit int) (result []testkube.Execution, err error) {
	result = make([]testkube.Execution, 0)
	resultLimit := int64(limit)
	opts := &options.FindOptions{Limit: &resultLimit}
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})

	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.ResultsColl.Find(ctx, bson.M{}, opts)
	if err != nil {
		return result, err
	}
	err = cursor.All(ctx, &result)

	for i := range result {
		result[i].UnscapeDots()
	}
	return
}

func (r *MongoRepository) GetExecutions(ctx context.Context, filter Filter) (result []testkube.Execution, err error) {
	result = make([]testkube.Execution, 0)
	query, opts := composeQueryAndOpts(filter)
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.ResultsColl.Find(ctx, query, opts)
	if err != nil {
		return
	}
	err = cursor.All(ctx, &result)

	for i := range result {
		result[i].UnscapeDots()
	}
	return
}

func (r *MongoRepository) Count(ctx context.Context, filter Filter) (count int64, err error) {
	query, _ := composeQueryAndOpts(filter)
	return r.ResultsColl.CountDocuments(ctx, query)
}

func (r *MongoRepository) GetExecutionTotals(ctx context.Context, paging bool, filter ...Filter) (totals testkube.ExecutionsTotals, err error) {
	var result []struct {
		Status string `bson:"_id"`
		Count  int    `bson:"count"`
	}

	query := bson.M{}
	if len(filter) > 0 {
		query, _ = composeQueryAndOpts(filter[0])
	}

	pipeline := []bson.D{
		{{Key: "$sort", Value: bson.M{"executionresult.status": 1}}},
		{{Key: "$match", Value: query}},
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "starttime", Value: -1}}}})
		if paging {
			pipeline = append(pipeline, bson.D{{Key: "$skip", Value: int64(filter[0].Page() * filter[0].PageSize())}})
			pipeline = append(pipeline, bson.D{{Key: "$limit", Value: int64(filter[0].PageSize())}})
		}
	} else {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "executionresult.status", Value: 1}}}})
	}

	pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$executionresult.status"},
		{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}})

	opts := options.Aggregate()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.ResultsColl.Aggregate(ctx, pipeline, opts)
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

func (r *MongoRepository) GetLabels(ctx context.Context) (labels map[string][]string, err error) {
	var result []struct {
		Labels bson.M `bson:"labels"`
	}

	opts := options.Find()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.ResultsColl.Find(ctx, bson.M{}, opts)
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

func (r *MongoRepository) Insert(ctx context.Context, result testkube.Execution) (err error) {
	output := result.ExecutionResult.Output
	result.ExecutionResult.Output = ""
	result.EscapeDots()
	_, err = r.ResultsColl.InsertOne(ctx, result)
	if err != nil {
		return
	}

	if !r.features.LogsV2 {
		err = r.OutputRepository.InsertOutput(ctx, result.Id, result.TestName, result.TestSuiteName, output)
	}
	return
}

func (r *MongoRepository) Update(ctx context.Context, result testkube.Execution) (err error) {
	output := result.ExecutionResult.Output
	result.ExecutionResult.Output = ""
	result.EscapeDots()
	_, err = r.ResultsColl.ReplaceOne(ctx, bson.M{"id": result.Id}, result)
	if err != nil {
		return
	}

	if !r.features.LogsV2 {
		err = r.OutputRepository.UpdateOutput(ctx, result.Id, result.TestName, result.TestSuiteName, output)
	}
	return
}

func (r *MongoRepository) UpdateResult(ctx context.Context, id string, result testkube.Execution) (err error) {
	output := result.ExecutionResult.Output
	result.ExecutionResult = result.ExecutionResult.GetDeepCopy()
	result.ExecutionResult.Output = ""
	result.ExecutionResult.Steps = cleanSteps(result.ExecutionResult.Steps)

	var execution testkube.Execution
	err = r.ResultsColl.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&execution)
	if err != nil {
		return err
	}

	errorMessage := ""
	if execution.ExecutionResult != nil {
		errorMessage = execution.ExecutionResult.ErrorMessage
	}

	if errorMessage != "" && result.ExecutionResult.ErrorMessage != "" {
		errorMessage += "\n"
	}

	result.ExecutionResult.ErrorMessage = errorMessage + result.ExecutionResult.ErrorMessage
	_, err = r.ResultsColl.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"executionresult": result.ExecutionResult}})
	if err != nil {
		return err
	}

	if !r.features.LogsV2 {
		err = r.OutputRepository.UpdateOutput(ctx, id, result.TestName, result.TestSuiteName, cleanOutput(output))
	}
	return
}

// StartExecution updates execution start time
func (r *MongoRepository) StartExecution(ctx context.Context, id string, startTime time.Time) (err error) {
	_, err = r.ResultsColl.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"starttime": startTime}})
	return
}

// EndExecution updates execution end time
func (r *MongoRepository) EndExecution(ctx context.Context, e testkube.Execution) (err error) {
	_, err = r.ResultsColl.UpdateOne(ctx, bson.M{"id": e.Id}, bson.M{"$set": bson.M{"endtime": e.EndTime, "duration": e.Duration, "durationms": e.DurationMs}})
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
	err = r.OutputRepository.DeleteOutputByTest(ctx, testName)
	if err != nil {
		return
	}
	err = r.DeleteExecutionNumber(ctx, testName)
	if err != nil {
		return
	}
	_, err = r.ResultsColl.DeleteMany(ctx, bson.M{"testname": testName})
	return
}

// DeleteByTestSuite deletes execution results by test suite
func (r *MongoRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) (err error) {
	err = r.OutputRepository.DeleteOutputByTestSuite(ctx, testSuiteName)
	if err != nil {
		return
	}
	err = r.DeleteExecutionNumber(ctx, testSuiteName)
	if err != nil {
		return
	}
	_, err = r.ResultsColl.DeleteMany(ctx, bson.M{"testsuitename": testSuiteName})
	return
}

// DeleteAll deletes all execution results
func (r *MongoRepository) DeleteAll(ctx context.Context) (err error) {
	err = r.OutputRepository.DeleteAllOutput(ctx)
	if err != nil {
		return
	}
	err = r.DeleteAllExecutionNumbers(ctx, false)
	if err != nil {
		return
	}
	_, err = r.ResultsColl.DeleteMany(ctx, bson.M{})
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

	err = r.OutputRepository.DeleteOutputForTests(ctx, testNames)
	if err != nil {
		return
	}

	err = r.DeleteExecutionNumbers(ctx, testNames)
	if err != nil {
		return
	}
	_, err = r.ResultsColl.DeleteMany(ctx, filter)
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

	err = r.OutputRepository.DeleteOutputForTestSuites(ctx, testSuiteNames)
	if err != nil {
		return
	}

	err = r.DeleteExecutionNumbers(ctx, testSuiteNames)
	if err != nil {
		return
	}

	_, err = r.ResultsColl.DeleteMany(ctx, filter)
	return
}

// DeleteForAllTestSuites deletes execution results for all test suites
func (r *MongoRepository) DeleteForAllTestSuites(ctx context.Context) (err error) {
	err = r.OutputRepository.DeleteOutputForAllTestSuite(ctx)
	if err != nil {
		return
	}

	err = r.DeleteAllExecutionNumbers(ctx, true)
	if err != nil {
		return
	}

	_, err = r.ResultsColl.DeleteMany(ctx, bson.M{"testsuitename": bson.M{"$ne": ""}})
	return
}

// GetTestMetrics returns test executions metrics limited to number of executions or last N days
func (r *MongoRepository) GetTestMetrics(ctx context.Context, name string, limit, last int) (metrics testkube.ExecutionsMetrics, err error) {
	query := bson.M{"testname": name}

	pipeline := []bson.D{}
	if last > 0 {
		query["starttime"] = bson.M{"$gte": time.Now().Add(-time.Duration(last) * 24 * time.Hour)}
	}

	pipeline = append(pipeline, bson.D{{Key: "$match", Value: query}})
	pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "starttime", Value: -1}}}})
	pipeline = append(pipeline, bson.D{
		{
			Key: "$project", Value: bson.D{
				{Key: "status", Value: "$executionresult.status"},
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

	cursor, err := r.ResultsColl.Aggregate(ctx, pipeline, opts)
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

// cleanOutput makes sure the output fits into the limits imposed by Mongo;
// if needed it trims the string
// it keeps the first OutputPrefixSize of strings in case there were errors on init
// it adds a warning that the logs were trimmed
// it adds the last OutputMaxSize-OutputPrefixSize-OverflownOutputWarnSize bytes to the end
func cleanOutput(output string) string {
	if len(output) >= OutputMaxSize {
		prefix := output[:OutputPrefixSize]
		suffix := output[len(output)-OutputMaxSize+OutputPrefixSize+len(OverflownOutputWarn):]
		output = fmt.Sprintf("%s\n%s\n%s", prefix, OverflownOutputWarn, suffix)
	}
	return output
}

// cleanSteps trims the list of ExecutionStepResults in case there's too many elements to make sure it fits into mongo
func cleanSteps(steps []testkube.ExecutionStepResult) []testkube.ExecutionStepResult {
	l := len(steps)
	if l > StepMaxCount {
		steps = steps[l-StepMaxCount:]
	}
	return steps
}
