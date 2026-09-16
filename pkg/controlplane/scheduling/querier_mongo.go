package scheduling

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// ExecutionQuerier accesses the underlying mongo database and queries a test workflow execution
// collection to gather information about executions that should have their state modified
// by a runner in some way.
// If either the database or collection name retrieval function are `nil` then no executions
// will ever be yielded by the iterator functions.
type MongoExecutionQuerier struct {
	executionsCollection *mongo.Collection
	byStatusPager        executionBatchPager[testkube.TestWorkflowExecution]
}

func NewMongoExecutionQuerier(col *mongo.Collection) *MongoExecutionQuerier {
	return &MongoExecutionQuerier{executionsCollection: col}
}

// Pausing yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be paused by the runner.
func (a MongoExecutionQuerier) Pausing(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"result.status": testkube.PAUSING_TestWorkflowStatus})
}

// Resuming yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be resumed by the runner.
func (a MongoExecutionQuerier) Resuming(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"result.status": testkube.RESUMING_TestWorkflowStatus})
}

// Aborting yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be aborted by the runner.
func (a MongoExecutionQuerier) Aborting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"$and": bson.A{
		bson.M{"result.status": testkube.STOPPING_TestWorkflowStatus},
		bson.M{"result.predictedstatus": bson.M{"$ne": testkube.CANCELED_TestWorkflowStatus}},
	}})
}

// Cancelling yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be cancelled by the runner.
func (a MongoExecutionQuerier) Cancelling(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"$and": bson.A{
		bson.M{"result.status": testkube.STOPPING_TestWorkflowStatus},
		bson.M{"result.predictedstatus": testkube.CANCELED_TestWorkflowStatus},
	}})
}

// Assigned yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be started by the runner.
func (a MongoExecutionQuerier) Assigned(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"result.status": testkube.ASSIGNED_TestWorkflowStatus})
}

// Starting yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be started by the runner.
func (a MongoExecutionQuerier) Starting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"result.status": testkube.STARTING_TestWorkflowStatus})
}

// ByStatus yields an iterator returning all executions that match one of the given statuses.
func (a *MongoExecutionQuerier) ByStatus(ctx context.Context, statuses []testkube.TestWorkflowStatus) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return func(yield func(testkube.TestWorkflowExecution, error) bool) {
		executions, err := a.byStatusPager.Next(
			func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
				match := bson.M{
					"result.status": bson.M{"$in": statuses},
					"pendingAt":     bson.M{"$lte": snapshotBefore},
				}
				if after != nil {
					match["$or"] = bson.A{
						bson.M{"pendingAt": bson.M{"$gt": after.pendingAt}},
						bson.M{
							"pendingAt": after.pendingAt,
							"id":        bson.M{"$gt": after.executionID},
						},
					}
				}

				cur, err := a.executionsCollection.Aggregate(ctx, mongo.Pipeline{
					{{Key: "$match", Value: bson.M{"result.status": bson.M{"$in": statuses}}}},
					{{Key: "$addFields", Value: bson.M{
						"pendingAt": bson.M{"$ifNull": bson.A{"$statusat", "$scheduledat"}},
					}}},
					{{Key: "$match", Value: match}},
					{{Key: "$sort", Value: bson.D{
						{Key: "pendingAt", Value: 1},
						{Key: "id", Value: 1},
					}}},
					{{Key: "$limit", Value: int64(executionUpdatesBatchSize)}},
				})
				if err != nil {
					return nil, err
				}
				defer func() {
					_ = cur.Close(ctx)
				}()

				var executions []testkube.TestWorkflowExecution
				for cur.Next(ctx) {
					var exe testkube.TestWorkflowExecution
					if err := cur.Decode(&exe); err != nil {
						return nil, err
					}
					executions = append(executions, exe)
				}
				return executions, cur.Err()
			},
			func(exe testkube.TestWorkflowExecution) *executionBatchCursor {
				pendingAt := exe.StatusAt
				if pendingAt.IsZero() {
					pendingAt = exe.ScheduledAt
				}
				return &executionBatchCursor{
					pendingAt:   pendingAt,
					executionID: exe.Id,
				}
			},
		)
		if err != nil {
			yield(testkube.TestWorkflowExecution{}, fmt.Errorf("find executions with ExecutionQuerier statuses: %w", err))
			return
		}
		for _, exe := range executions {
			if !yield(exe, nil) {
				return
			}
		}
	}
}

func (a MongoExecutionQuerier) executionIterator(ctx context.Context, filter any) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return func(yield func(testkube.TestWorkflowExecution, error) bool) {
		cur, err := a.executionsCollection.Find(ctx, filter)
		if err != nil {
			yield(testkube.TestWorkflowExecution{}, fmt.Errorf("find executions with ExecutionQuerier statuses: %w", err))
			return
		}
		defer func() {
			if err := cur.Close(ctx); err != nil {
				yield(testkube.TestWorkflowExecution{}, fmt.Errorf("close cursor: %w", err))
			}
		}()
		for cur.Next(ctx) {
			var exe testkube.TestWorkflowExecution
			if err := cur.Decode(&exe); err != nil {
				if !yield(exe, fmt.Errorf("decode test workflow execution: %w", err)) {
					return
				}
				continue
			}
			if !yield(exe, nil) {
				return
			}
		}
		if err := cur.Err(); err != nil {
			yield(testkube.TestWorkflowExecution{}, fmt.Errorf("cursor error: %w", err))
		}
	}
}
