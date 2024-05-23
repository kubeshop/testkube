// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflow

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

const CollectionName = "testworkflowresults"

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
	db           *mongo.Database
	Coll         *mongo.Collection
	allowDiskUse bool
}

func WithMongoRepositoryCollection(collection *mongo.Collection) MongoRepositoryOpt {
	return func(r *MongoRepository) {
		r.Coll = collection
	}
}

type MongoRepositoryOpt func(*MongoRepository)

func (r *MongoRepository) Get(ctx context.Context, id string) (result testkube.TestWorkflowExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&result)
	return *result.UnscapeDots(), err
}

func (r *MongoRepository) GetByNameAndTestWorkflow(ctx context.Context, name, workflowName string) (result testkube.TestWorkflowExecution, err error) {
	err = r.Coll.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": name}, bson.M{"name": name}}, "workflow.name": workflowName}).Decode(&result)
	return *result.UnscapeDots(), err
}

func (r *MongoRepository) GetLatestByTestWorkflow(ctx context.Context, workflowName string) (*testkube.TestWorkflowExecution, error) {
	opts := options.Aggregate()
	pipeline := []bson.M{
		{"$sort": bson.M{"statusat": -1}},
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
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "statusat", Value: -1}}}})
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: int64(filter[0].Page() * filter[0].PageSize())}})
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

func (r *MongoRepository) GetExecutionsSummary(ctx context.Context, filter Filter) (result []testkube.TestWorkflowExecutionSummary, err error) {
	result = make([]testkube.TestWorkflowExecutionSummary, 0)
	query, opts := composeQueryAndOpts(filter)
	if r.allowDiskUse {
		opts.SetAllowDiskUse(r.allowDiskUse)
	}

	opts = opts.SetProjection(bson.M{
		"_id":                   0,
		"output":                0,
		"signature":             0,
		"result.steps":          0,
		"result.initialization": 0,
		"workflow.spec":         0,
		"resolvedWorkflow":      0,
	})
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

func (r *MongoRepository) Insert(ctx context.Context, result testkube.TestWorkflowExecution) (err error) {
	result.EscapeDots()
	if result.Reports == nil {
		result.Reports = []testkube.TestWorkflowReport{}
	}
	_, err = r.Coll.InsertOne(ctx, result)
	return
}

func (r *MongoRepository) Update(ctx context.Context, result testkube.TestWorkflowExecution) (err error) {
	result.EscapeDots()
	_, err = r.Coll.ReplaceOne(ctx, bson.M{"id": result.Id}, result)
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

func composeQueryAndOpts(filter Filter) (bson.M, *options.FindOptions) {
	query := bson.M{}
	opts := options.Find()
	startTimeQuery := bson.M{}

	if filter.NameDefined() {
		query["workflow.name"] = filter.Name()
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
		if len(statuses) == 1 {
			query["result.status"] = statuses[0]
		} else {
			var conditions bson.A
			for _, status := range statuses {
				conditions = append(conditions, bson.M{"result.status": status})
			}

			query["$or"] = conditions
		}
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

	opts.SetSkip(int64(filter.Page() * filter.PageSize()))
	opts.SetLimit(int64(filter.PageSize()))
	opts.SetSort(bson.D{{Key: "scheduledat", Value: -1}})

	return query, opts
}

// DeleteByTestWorkflow deletes execution results by workflow
func (r *MongoRepository) DeleteByTestWorkflow(ctx context.Context, workflowName string) (err error) {
	_, err = r.Coll.DeleteMany(ctx, bson.M{"workflow.name": workflowName})
	return
}

// DeleteAll deletes all execution results
func (r *MongoRepository) DeleteAll(ctx context.Context) (err error) {
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
			"status":    "$result.status",
			"duration":  "$result.duration",
			"starttime": "$scheduledat",
			"name":      1,
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
