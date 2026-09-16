package scheduling

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// ExecutionQuerier accesses the underlying mongo database and queries a test workflow execution
// collection to gather information about executions that should have their state modified
// by a runner in some way.
// If either the database or collection name retrieval function are `nil` then no executions
// will ever be yielded by the iterator functions.
type MongoExecutionQuerier struct {
	executionsCollection *mongo.Collection
	allowDiskUse         bool
	byStatusPager        executionBatchPager[testkube.TestWorkflowExecution]
}

func NewMongoExecutionQuerier(col *mongo.Collection, allowDiskUse bool) *MongoExecutionQuerier {
	return &MongoExecutionQuerier{executionsCollection: col, allowDiskUse: allowDiskUse}
}

// Pausing yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be paused by the runner.
func (a *MongoExecutionQuerier) Pausing(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"result.status": testkube.PAUSING_TestWorkflowStatus})
}

// Resuming yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be resumed by the runner.
func (a *MongoExecutionQuerier) Resuming(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"result.status": testkube.RESUMING_TestWorkflowStatus})
}

// Aborting yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be aborted by the runner.
func (a *MongoExecutionQuerier) Aborting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"$and": bson.A{
		bson.M{"result.status": testkube.STOPPING_TestWorkflowStatus},
		bson.M{"result.predictedstatus": bson.M{"$ne": testkube.CANCELED_TestWorkflowStatus}},
	}})
}

// Cancelling yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be cancelled by the runner.
func (a *MongoExecutionQuerier) Cancelling(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"$and": bson.A{
		bson.M{"result.status": testkube.STOPPING_TestWorkflowStatus},
		bson.M{"result.predictedstatus": testkube.CANCELED_TestWorkflowStatus},
	}})
}

// Assigned yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be started by the runner.
func (a *MongoExecutionQuerier) Assigned(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"result.status": testkube.ASSIGNED_TestWorkflowStatus})
}

// Starting yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be started by the runner.
func (a *MongoExecutionQuerier) Starting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return a.executionIterator(ctx, bson.M{"result.status": testkube.STARTING_TestWorkflowStatus})
}

// ByStatus yields an iterator returning all executions that match one of the given statuses.
func (a *MongoExecutionQuerier) ByStatus(ctx context.Context, statuses []testkube.TestWorkflowStatus) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	return func(yield func(testkube.TestWorkflowExecution, error) bool) {
		executions, err := a.byStatusPager.Next(
			func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
				withStatusAt, err := a.findPendingExecutions(ctx, statuses, snapshotBefore, after, "statusat", true)
				if err != nil {
					return nil, err
				}

				withScheduledAt, err := a.findPendingExecutions(ctx, statuses, snapshotBefore, after, "scheduledat", false)
				if err != nil {
					return nil, err
				}

				return mergePendingExecutions(withStatusAt, withScheduledAt, executionUpdatesBatchSize), nil
			},
			func(exe testkube.TestWorkflowExecution) *executionBatchCursor {
				return &executionBatchCursor{
					pendingAt:   pendingExecutionTime(exe),
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

func (a *MongoExecutionQuerier) findPendingExecutions(
	ctx context.Context,
	statuses []testkube.TestWorkflowStatus,
	snapshotBefore time.Time,
	after *executionBatchCursor,
	pendingField string,
	hasStatusAt bool,
) ([]testkube.TestWorkflowExecution, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: pendingField, Value: 1}, {Key: "id", Value: 1}}).
		SetLimit(int64(executionUpdatesBatchSize))
	if a.allowDiskUse {
		opts.SetAllowDiskUse(true)
	}

	cur, err := a.executionsCollection.Find(ctx, pendingExecutionFilter(statuses, snapshotBefore, after, pendingField, hasStatusAt), opts)
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
}

func pendingExecutionFilter(
	statuses []testkube.TestWorkflowStatus,
	snapshotBefore time.Time,
	after *executionBatchCursor,
	pendingField string,
	hasStatusAt bool,
) bson.M {
	clauses := bson.A{
		bson.M{"result.status": bson.M{"$in": statuses}},
		bson.M{pendingField: bson.M{"$lte": snapshotBefore}},
	}

	if hasStatusAt {
		clauses = append(clauses, bson.M{"statusat": bson.M{"$gt": time.Time{}}})
	} else {
		clauses = append(clauses, bson.M{"$or": bson.A{
			bson.M{"statusat": bson.M{"$exists": false}},
			bson.M{"statusat": nil},
			bson.M{"statusat": time.Time{}},
		}})
	}

	if after != nil {
		clauses = append(clauses, bson.M{"$or": bson.A{
			bson.M{pendingField: bson.M{"$gt": after.pendingAt}},
			bson.M{
				pendingField: after.pendingAt,
				"id":         bson.M{"$gt": after.executionID},
			},
		}})
	}

	return bson.M{"$and": clauses}
}

func mergePendingExecutions(withStatusAt, withScheduledAt []testkube.TestWorkflowExecution, limit int) []testkube.TestWorkflowExecution {
	executions := make([]testkube.TestWorkflowExecution, 0, limit)
	i, j := 0, 0
	for len(executions) < limit && (i < len(withStatusAt) || j < len(withScheduledAt)) {
		switch {
		case j >= len(withScheduledAt):
			executions = append(executions, withStatusAt[i])
			i++
		case i >= len(withStatusAt):
			executions = append(executions, withScheduledAt[j])
			j++
		case pendingExecutionLess(withStatusAt[i], withScheduledAt[j]):
			executions = append(executions, withStatusAt[i])
			i++
		default:
			executions = append(executions, withScheduledAt[j])
			j++
		}
	}
	return executions
}

func pendingExecutionLess(left, right testkube.TestWorkflowExecution) bool {
	leftPendingAt := pendingExecutionTime(left)
	rightPendingAt := pendingExecutionTime(right)
	if leftPendingAt.Equal(rightPendingAt) {
		return left.Id < right.Id
	}
	return leftPendingAt.Before(rightPendingAt)
}

func pendingExecutionTime(exe testkube.TestWorkflowExecution) time.Time {
	if !exe.StatusAt.IsZero() {
		return exe.StatusAt
	}
	return exe.ScheduledAt
}

func (a *MongoExecutionQuerier) executionIterator(ctx context.Context, filter any) func(yield func(testkube.TestWorkflowExecution, error) bool) {
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
