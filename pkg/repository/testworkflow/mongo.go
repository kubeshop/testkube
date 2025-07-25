package testworkflow

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/repository/common"
	"github.com/kubeshop/testkube/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
)

var _ Repository = (*MongoRepository)(nil)

const (
	CollectionName       = "testworkflowresults"
	configParamSizeLimit = 100
)

func NewMongoRepository(db *mongo.Database, allowDiskUse bool, opts ...MongoRepositoryOpt) *MongoRepository {
	r := &MongoRepository{
		db:           db,
		Coll:         db.Collection(CollectionName),
		allowDiskUse: allowDiskUse,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

type MongoRepository struct {
	db                 *mongo.Database
	Coll               *mongo.Collection
	allowDiskUse       bool
	sequenceRepository sequence.Repository
}

func WithMongoRepositoryCollection(collection *mongo.Collection) MongoRepositoryOpt {
	return func(r *MongoRepository) {
		r.Coll = collection
	}
}

func WithMongoRepositorySequence(sequenceRepository sequence.Repository) MongoRepositoryOpt {
	return func(r *MongoRepository) {
		r.sequenceRepository = sequenceRepository
	}
}

type MongoRepositoryOpt func(*MongoRepository)

func (r *MongoRepository) Get(ctx context.Context, id string) (result testkube.TestWorkflowExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&result)

	if result.ResolvedWorkflow != nil && result.ResolvedWorkflow.Spec != nil {
		result.ConfigParams = populateConfigParams(result.ResolvedWorkflow, result.ConfigParams)
	}

	return *result.UnscapeDots(), err
}

func populateConfigParams(resolvedWorkflow *testkube.TestWorkflow, configParams map[string]testkube.TestWorkflowExecutionConfigValue) map[string]testkube.TestWorkflowExecutionConfigValue {
	if configParams == nil {
		configParams = make(map[string]testkube.TestWorkflowExecutionConfigValue)
	}

	for k, v := range resolvedWorkflow.Spec.Config {
		if v.Sensitive {
			configParams[k] = testkube.TestWorkflowExecutionConfigValue{
				Sensitive:         true,
				EmptyValue:        true,
				EmptyDefaultValue: true,
			}

			continue
		}

		if _, ok := configParams[k]; !ok {
			configParams[k] = testkube.TestWorkflowExecutionConfigValue{
				EmptyValue: true,
			}
		}

		data := configParams[k]
		if len(data.Value) > configParamSizeLimit {
			data.Value = data.Value[:configParamSizeLimit]
			data.Truncated = true
		}

		if v.Default_ != nil {
			data.DefaultValue = v.Default_.Value
		} else {
			data.EmptyDefaultValue = true
		}

		configParams[k] = data
	}

	return configParams
}

func (r *MongoRepository) GetByNameAndTestWorkflow(ctx context.Context, name, workflowName string) (result testkube.TestWorkflowExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": name}, bson.M{"name": name}}, "workflow.name": workflowName}).Decode(&result)
	return *result.UnscapeDots(), err
}

// GetLatestByTestWorkflow retrieves the latest test workflow execution for a given workflow name with configurable sorting
func (r *MongoRepository) GetLatestByTestWorkflow(ctx context.Context, workflowName string, sortBy LatestSortBy) (*testkube.TestWorkflowExecution, error) {
	sortField := "statusat"
	if sortBy == LatestSortByNumber {
		sortField = "number"
	}

	opts := options.Aggregate()
	pipeline := []bson.M{
		{"$sort": bson.M{sortField: -1}},
		{"$match": bson.M{"workflow.name": workflowName}},
		{"$limit": 1},
	}
	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	var items []testkube.TestWorkflowExecution
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return items[0].UnscapeDots(), err
}

func (r *MongoRepository) GetLatestByTestWorkflows(ctx context.Context, workflowNames []string) (executions []testkube.TestWorkflowExecutionSummary, err error) {
	if len(workflowNames) == 0 {
		return executions, nil
	}

	documents := bson.A{}
	for _, workflowName := range workflowNames {
		documents = append(documents, bson.M{"workflow.name": workflowName})
	}

	pipeline := []bson.M{
		{"$sort": bson.M{"statusat": -1}},
		{"$project": bson.M{
			"_id":                   0,
			"output":                0,
			"signature":             0,
			"result.steps":          0,
			"result.initialization": 0,
			"workflow.spec":         0,
			"resolvedWorkflow":      0,
		}},
		{"$match": bson.M{"$or": documents}},
		{"$group": bson.M{"_id": "$workflow.name", "execution": bson.M{"$first": "$$ROOT"}}},
		{"$replaceRoot": bson.M{"newRoot": "$execution"}},
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

// TODO: Add limit?
func (r *MongoRepository) GetRunning(ctx context.Context) (result []testkube.TestWorkflowExecution, err error) {
	result = make([]testkube.TestWorkflowExecution, 0)
	opts := &options.FindOptions{}
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Find(ctx, bson.M{
		"$or": bson.A{
			bson.M{"result.status": testkube.PAUSED_TestWorkflowStatus},
			bson.M{"result.status": testkube.RUNNING_TestWorkflowStatus},
			bson.M{"result.status": testkube.QUEUED_TestWorkflowStatus},
		},
	}, opts)
	if err != nil {
		return result, err
	}
	err = cursor.All(ctx, &result)

	for i := range result {
		result[i].UnscapeDots()
	}
	return
}

func (r *MongoRepository) GetFinished(ctx context.Context, filter Filter) (result []testkube.TestWorkflowExecution, err error) {
	result = make([]testkube.TestWorkflowExecution, 0)
	query, opts := composeQueryAndOpts(filter)
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}
	query["$or"] = bson.A{
		bson.M{"result.status": testkube.PASSED_TestWorkflowStatus},
		bson.M{"result.status": testkube.FAILED_TestWorkflowStatus},
		bson.M{"result.status": testkube.ABORTED_TestWorkflowStatus},
	}

	cursor, err := r.Coll.Find(ctx, query, opts)
	if err != nil {
		return result, err
	}
	err = cursor.All(ctx, &result)

	for i := range result {
		result[i].UnscapeDots()
	}
	return
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

	pipeline := []bson.D{{{Key: "$match", Value: query}}}
	hasSkip := len(filter) > 0 && filter[0].Page() > 0
	hasLimit := len(filter) > 0 && filter[0].PageSize() < math.MaxInt32
	if hasSkip || hasLimit {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "statusat", Value: -1}}}})
	}
	if hasSkip {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: int64(filter[0].Page() * filter[0].PageSize())}})
	}
	if hasLimit {
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: int64(filter[0].PageSize())}})
	}

	pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$result.status"},
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
		switch testkube.TestWorkflowStatus(o.Status) {
		case testkube.QUEUED_TestWorkflowStatus:
			totals.Queued = int32(o.Count)
		case testkube.RUNNING_TestWorkflowStatus:
			totals.Running = int32(o.Count)
		case testkube.PASSED_TestWorkflowStatus:
			totals.Passed = int32(o.Count)
		case testkube.FAILED_TestWorkflowStatus, testkube.ABORTED_TestWorkflowStatus:
			totals.Failed = int32(o.Count)
		}
	}
	totals.Results = sum

	return
}

func (r *MongoRepository) Count(ctx context.Context, filter Filter) (count int64, err error) {
	query, _ := composeQueryAndOpts(filter)
	return r.Coll.CountDocuments(ctx, query)
}

func (r *MongoRepository) GetExecutions(ctx context.Context, filter Filter) (result []testkube.TestWorkflowExecution, err error) {
	result = make([]testkube.TestWorkflowExecution, 0)
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

type TestWorkflowExecutionSummaryWithResolvedWorkflow struct {
	testkube.TestWorkflowExecutionSummary `json:",inline" bson:",inline"`
	ResolvedWorkflow                      *testkube.TestWorkflow `json:"resolvedWorkflow,omitempty"`
}

func (r *MongoRepository) GetExecutionsSummary(ctx context.Context, filter Filter) (result []testkube.TestWorkflowExecutionSummary, err error) {
	executions := make([]TestWorkflowExecutionSummaryWithResolvedWorkflow, 0)
	query, _ := composeQueryAndOpts(filter)

	pipeline := []bson.M{
		{"$sort": bson.M{"scheduledat": -1}},
		{"$match": query},
		{"$project": bson.M{
			"_id":           0,
			"workflow.spec": 0,
		}},
		{"$project": bson.M{
			"id":                           1,
			"groupid":                      1,
			"runnerid":                     1,
			"name":                         1,
			"number":                       1,
			"scheduledat":                  1,
			"statusat":                     1,
			"result":                       1,
			"workflow":                     1,
			"tags":                         1,
			"runningcontext":               1,
			"configparams":                 1,
			"resolvedworkflow.spec.config": 1,
			"reports":                      1,
			"resourceaggregations":         1,
		}},
	}

	if filter.PageSize() > 0 {
		if filter.Page() > 0 {
			pipeline = append(pipeline, bson.M{"$skip": int64(filter.Page() * filter.PageSize())})
		}
		pipeline = append(pipeline, bson.M{"$limit": int64(filter.PageSize())})
	}

	opts := options.Aggregate()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return
	}
	err = cursor.All(ctx, &executions)
	result = make([]testkube.TestWorkflowExecutionSummary, len(executions))
	for i := range executions {
		executions[i].UnscapeDots()

		if executions[i].ResolvedWorkflow != nil && executions[i].ResolvedWorkflow.Spec != nil {
			executions[i].ConfigParams = populateConfigParams(executions[i].ResolvedWorkflow, executions[i].ConfigParams)
		}
		result[i] = executions[i].TestWorkflowExecutionSummary
	}
	return
}

func (r *MongoRepository) Insert(ctx context.Context, result testkube.TestWorkflowExecution) (err error) {
	execution := result.Clone()
	execution.EscapeDots()
	if execution.Reports == nil {
		execution.Reports = []testkube.TestWorkflowReport{}
	}
	_, err = r.Coll.InsertOne(ctx, execution)
	return
}

func (r *MongoRepository) Update(ctx context.Context, result testkube.TestWorkflowExecution) (err error) {
	execution := result.Clone()
	execution.EscapeDots()
	if execution.Reports == nil {
		execution.Reports = []testkube.TestWorkflowReport{}
	}
	_, err = r.Coll.ReplaceOne(ctx, bson.M{"id": execution.Id}, execution)
	return
}

func (r *MongoRepository) UpdateResult(ctx context.Context, id string, result *testkube.TestWorkflowResult) (err error) {
	data := bson.M{"result": result}
	if !result.FinishedAt.IsZero() {
		data["statusat"] = result.FinishedAt
	}
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": data})
	return
}

func (r *MongoRepository) UpdateReport(ctx context.Context, id string, report *testkube.TestWorkflowReport) (err error) {
	filter := bson.M{"id": id}
	update := bson.M{"$push": bson.M{"reports": report}}

	_, err = r.Coll.UpdateOne(ctx, filter, update)
	return
}

func (r *MongoRepository) UpdateOutput(ctx context.Context, id string, refs []testkube.TestWorkflowOutput) (err error) {
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"output": refs}})
	return
}

func (r *MongoRepository) UpdateResourceAggregations(ctx context.Context, id string, resourceAggregations *testkube.TestWorkflowExecutionResourceAggregationsReport) (err error) {
	_, err = r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"resourceaggregations": resourceAggregations}})
	return
}

func composeQueryAndOpts(filter Filter) (bson.M, *options.FindOptions) {
	query := bson.M{}
	opts := options.Find()
	startTimeQuery := bson.M{}

	if filter.NameDefined() {
		query["workflow.name"] = filter.Name()
	}

	if filter.NamesDefined() {
		query["workflow.name"] = bson.M{"$in": filter.Names()}
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
		query["scheduledat"] = startTimeQuery
	}

	if filter.StatusesDefined() {
		statuses := filter.Statuses()
		query["result.status"] = bson.M{"$in": statuses}
	}

	if filter.Selector() != "" {
		items := strings.Split(filter.Selector(), ",")
		for _, item := range items {
			elements := strings.Split(item, "=")
			if len(elements) == 2 {
				query["workflow.labels."+elements[0]] = elements[1]
			} else if len(elements) == 1 {
				query["workflow.labels."+elements[0]] = bson.M{"$exists": true}
			}
		}
	}

	if filter.TagSelector() != "" {
		items := strings.Split(filter.TagSelector(), ",")
		inValues := make(map[string][]string)
		existsValues := make(map[string]struct{})
		for _, item := range items {
			elements := strings.Split(item, "=")
			if len(elements) == 2 {
				inValues["tags."+utils.EscapeDots(elements[0])] = append(inValues["tags."+utils.EscapeDots(elements[0])], elements[1])
			} else if len(elements) == 1 {
				existsValues["tags."+utils.EscapeDots(elements[0])] = struct{}{}
			}
		}
		subquery := bson.A{}
		for tag, values := range inValues {
			if _, ok := existsValues[tag]; ok {
				subquery = append(subquery, bson.M{tag: bson.M{"$exists": true}})
				delete(existsValues, tag)
				continue
			}

			tagValues := bson.A{}
			for _, value := range values {
				tagValues = append(tagValues, value)
			}

			if len(tagValues) > 0 {
				subquery = append(subquery, bson.M{tag: bson.M{"$in": tagValues}})
			}
		}

		for tag := range existsValues {
			subquery = append(subquery, bson.M{tag: bson.M{"$exists": true}})
		}

		if len(subquery) > 0 {
			query["$and"] = subquery
		}
	}

	if filter.LabelSelector() != nil && len(filter.LabelSelector().Or) > 0 {
		subquery := bson.A{}
		for _, label := range filter.LabelSelector().Or {
			if label.Value != nil {
				subquery = append(subquery, bson.M{"workflow.labels." + utils.EscapeDots(label.Key): *label.Value})
			} else if label.Exists != nil {
				subquery = append(subquery,
					bson.M{"workflow.labels." + utils.EscapeDots(label.Key): bson.M{"$exists": *label.Exists}})
			}
		}
		query["$or"] = subquery
	}

	if filter.ActorNameDefined() {
		query["runningcontext.actor.name"] = filter.ActorName()
	}

	if filter.ActorTypeDefined() {
		query["runningcontext.actor.type_"] = filter.ActorType()
	}

	if filter.RunnerIDDefined() {
		query["runnerid"] = filter.RunnerID()
	} else if filter.AssignedDefined() {
		if filter.Assigned() {
			query["runnerid"] = bson.M{"$not": bson.M{"$in": bson.A{nil, ""}}}
		} else {
			query["runnerid"] = bson.M{"$in": bson.A{nil, ""}}
		}
	}

	if filter.InitializedDefined() {
		var q bson.M
		if filter.Initialized() {
			q = bson.M{"$expr": bson.M{"$or": bson.A{
				bson.M{"$ne": bson.A{"$result.status", "queued"}},
				bson.M{"$and": []bson.M{
					{"$not": bson.M{"$in": bson.A{"$result.steps", bson.A{nil, bson.M{}}}}},
				}},
			}}}
		} else {
			q = bson.M{"$expr": bson.M{"$and": bson.A{
				bson.M{"$eq": bson.A{"$result.status", "queued"}},
				bson.M{"$in": bson.A{"$result.steps", bson.A{nil, bson.M{}}}},
			}}}
		}
		query = bson.M{"$and": bson.A{query, q}}
	}

	if filter.GroupIDDefined() {
		query = bson.M{"$and": bson.A{
			bson.M{"$expr": bson.M{"$or": bson.A{
				bson.M{"$eq": bson.A{"$id", filter.GroupID()}},
				bson.M{"$eq": bson.A{"$groupid", filter.GroupID()}},
			}}},
			query,
		}}
	}

	opts.SetSkip(int64(filter.Page() * filter.PageSize()))
	opts.SetLimit(int64(filter.PageSize()))
	opts.SetSort(bson.D{{Key: "scheduledat", Value: -1}})

	return query, opts
}

// DeleteByTestWorkflow deletes execution results by workflow
func (r *MongoRepository) DeleteByTestWorkflow(ctx context.Context, workflowName string) (err error) {
	if r.sequenceRepository != nil {
		err = r.sequenceRepository.DeleteExecutionNumber(ctx, workflowName, sequence.ExecutionTypeTestWorkflow)
		if err != nil {
			return
		}
	}

	_, err = r.Coll.DeleteMany(ctx, bson.M{"workflow.name": workflowName})
	return
}

// DeleteAll deletes all execution results
func (r *MongoRepository) DeleteAll(ctx context.Context) (err error) {
	if r.sequenceRepository != nil {
		err = r.sequenceRepository.DeleteAllExecutionNumbers(ctx, sequence.ExecutionTypeTestWorkflow)
		if err != nil {
			return
		}
	}

	_, err = r.Coll.DeleteMany(ctx, bson.M{})
	return
}

// DeleteByTestWorkflows deletes execution results by workflows
func (r *MongoRepository) DeleteByTestWorkflows(ctx context.Context, workflowNames []string) (err error) {
	if len(workflowNames) == 0 {
		return nil
	}

	conditions := bson.A{}
	for _, workflowName := range workflowNames {
		conditions = append(conditions, bson.M{"workflow.name": workflowName})
	}

	filter := bson.M{"$or": conditions}

	if r.sequenceRepository != nil {
		err = r.sequenceRepository.DeleteExecutionNumbers(ctx, workflowNames, sequence.ExecutionTypeTestSuite)
		if err != nil {
			return
		}
	}

	_, err = r.Coll.DeleteMany(ctx, filter)
	return
}

// TODO: Avoid calculating for all executions in memory (same for tests/test suites)
// GetTestWorkflowMetrics returns test executions metrics
func (r *MongoRepository) GetTestWorkflowMetrics(ctx context.Context, name string, limit, last int) (metrics testkube.ExecutionsMetrics, err error) {
	query := bson.M{"workflow.name": name}

	if last > 0 {
		query["scheduledat"] = bson.M{"$gte": time.Now().Add(-time.Duration(last) * 24 * time.Hour)}
	}

	pipeline := []bson.M{
		{"$sort": bson.M{"scheduledat": -1}},
		{"$match": query},
		{"$project": bson.M{
			"_id":         0,
			"executionid": "$id",
			"groupid":     1,
			"duration":    "$result.duration",
			"durationms":  "$result.durationms",
			"status":      "$result.status",
			"name":        1,
			"starttime":   "$scheduledat",
			"runnerid":    1,
		}},
	}

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

// GetPreviousFinishedState gets previous finished execution state by test workflow
func (r *MongoRepository) GetPreviousFinishedState(ctx context.Context, testWorkflowName string, date time.Time) (testkube.TestWorkflowStatus, error) {
	opts := options.FindOne().SetProjection(bson.M{"result.status": 1}).SetSort(bson.D{{Key: "result.finishedat", Value: -1}})
	filter := bson.D{
		{Key: "workflow.name", Value: testWorkflowName},
		{Key: "result.finishedat", Value: bson.M{"$lt": date}},
		{Key: "result.status", Value: bson.M{"$in": []string{"passed", "failed", "skipped", "aborted", "canceled", "timeout"}}},
	}

	var result testkube.TestWorkflowExecution
	err := r.Coll.FindOne(ctx, filter, opts).Decode(&result)
	if err != nil && err == mongo.ErrNoDocuments {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("error decoding previous finished execution status: %w", err)
	}

	if result.Result == nil || result.Result.Status == nil {
		return "", nil
	}

	return *result.Result.Status, nil
}

// GetNextExecutionNumber gets next execution number by name
func (r *MongoRepository) GetNextExecutionNumber(ctx context.Context, name string) (number int32, err error) {
	if r.sequenceRepository == nil {
		return 0, errors.New("no sequence repository provided")
	}

	return r.sequenceRepository.GetNextExecutionNumber(ctx, name, sequence.ExecutionTypeTestWorkflow)
}

func (r *MongoRepository) GetExecutionTags(ctx context.Context, testWorkflowName string) (tags map[string][]string, err error) {
	query := bson.M{"tags": bson.M{"$nin": bson.A{nil, bson.M{}}}}
	if testWorkflowName != "" {
		query["workflow.name"] = testWorkflowName
	}

	pipeline := []bson.M{
		{"$match": query},
		{"$project": bson.M{"_id": 0, "tags": bson.M{"$objectToArray": "$tags"}}},
		{"$unwind": "$tags"},
		{"$group": bson.M{"_id": "$tags.k", "values": bson.M{"$addToSet": "$tags.v"}}},
		{"$project": bson.M{"_id": 0, "name": "$_id", "values": 1}},
	}

	opts := options.Aggregate()
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}

	var res []struct {
		Name   string   `bson:"name"`
		Values []string `bson:"values"`
	}
	err = cursor.All(ctx, &res)
	if err != nil {
		return nil, err
	}

	tags = make(map[string][]string)
	for _, tag := range res {
		tags[tag.Name] = tag.Values
	}

	return tags, nil
}

func (r *MongoRepository) Init(ctx context.Context, id string, data InitData) error {
	_, err := r.Coll.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": map[string]interface{}{
		"namespace": data.Namespace,
		"signature": data.Signature,
		"runnerid":  data.RunnerID,
	}})
	return err
}

func (r *MongoRepository) Assign(ctx context.Context, id string, prevRunnerId string, newRunnerId string, assignedAt *time.Time) (bool, error) {
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	res, err := r.Coll.UpdateOne(ctx, bson.M{
		"$and": []bson.M{
			{"id": id},
			{"result.status": testkube.QUEUED_TestWorkflowStatus},
			{"$or": []bson.M{
				// New assignment - workflow has no runner assigned
				{
					"$or": []bson.M{
						{"runnerid": nil},
						{"runnerid": ""},
					},
				},
				// Extension of existing assignment - extension to assignment timeout
				{
					"$and": []bson.M{
						{"runnerid": newRunnerId},
						{"assignedat": bson.M{"$lt": assignedAt}},
					},
				},
				// Reassignment to new runner - must wait one minute between assignments
				{
					"$and": []bson.M{
						{"runnerid": prevRunnerId},
						{"assignedat": bson.M{"$lt": oneMinuteAgo}},
						{"assignedat": bson.M{"$lt": assignedAt}},
					},
				},
			}},
		},
	}, bson.M{"$set": map[string]interface{}{
		"runnerid":   newRunnerId,
		"assignedat": assignedAt,
	}})
	if err != nil {
		return false, err
	}
	return res.MatchedCount > 0, nil
}

// TODO: Return IDs only
// TODO: Add indexes
func (r *MongoRepository) GetUnassigned(ctx context.Context) (result []testkube.TestWorkflowExecution, err error) {
	result = make([]testkube.TestWorkflowExecution, 0)
	opts := &options.FindOptions{}
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	cursor, err := r.Coll.Find(ctx, bson.M{
		"$and": []bson.M{
			{"result.status": testkube.QUEUED_TestWorkflowStatus},
			{"$or": []bson.M{{"runnerid": ""}, {"runnerid": nil}}},
		},
	}, opts)
	if err != nil {
		return result, err
	}
	err = cursor.All(ctx, &result)

	for i := range result {
		result[i].UnscapeDots()
	}
	return
}

func (r *MongoRepository) AbortIfQueued(ctx context.Context, id string) (ok bool, err error) {
	ts := time.Now()
	res, err := r.Coll.UpdateOne(ctx, bson.M{
		"$and": []bson.M{
			{"id": id},
			{"result.status": bson.M{"$in": bson.A{testkube.QUEUED_TestWorkflowStatus, testkube.RUNNING_TestWorkflowStatus, testkube.PAUSED_TestWorkflowStatus}}},
			{"$or": []bson.M{{"runnerid": ""}, {"runnerid": nil}}},
		},
	}, bson.M{"$set": map[string]interface{}{
		"result.status":                      testkube.ABORTED_TestWorkflowStatus,
		"result.predictedstatus":             testkube.ABORTED_TestWorkflowStatus,
		"statusat":                           ts,
		"result.finishedat":                  ts,
		"result.initialization.status":       testkube.ABORTED_TestWorkflowStatus,
		"result.initialization.errormessage": "Aborted before initialization.",
		"result.initialization.finishedat":   ts,
		//"result.totaldurationms": ts.Sub(scheduledAt).Milliseconds(),
	}})
	if err != nil {
		return false, err
	}
	return res.ModifiedCount > 0, nil
}
