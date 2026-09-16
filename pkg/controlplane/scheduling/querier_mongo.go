package scheduling

import (
	"context"
	"fmt"
	"sync"
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
	byStatusPager        mongoExecutionBatchPager
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
				return a.findPendingExecutions(ctx, statuses, snapshotBefore, after, "statusat", true)
			},
			func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
				return a.findPendingExecutions(ctx, statuses, snapshotBefore, after, "scheduledat", false)
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

type mongoExecutionBatchPager struct {
	mu                 sync.Mutex
	snapshotBefore     time.Time
	statusAtCursor     *executionBatchCursor
	scheduledAtCursor  *executionBatchCursor
	statusAtBuffer     []testkube.TestWorkflowExecution
	scheduledAtBuffer  []testkube.TestWorkflowExecution
	statusAtExhausted  bool
	scheduledExhausted bool
	now                func() time.Time
}

func (p *mongoExecutionBatchPager) Next(
	fetchStatusAt func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error),
	fetchScheduledAt func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error),
) ([]testkube.TestWorkflowExecution, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active() {
		p.resetSnapshot()
	}

	hadProgress := p.hasProgress()
	items, err := p.fetchPage(fetchStatusAt, fetchScheduledAt)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items, nil
	}
	if !hadProgress {
		p.clear()
		return nil, nil
	}

	p.resetSnapshot()
	items, err = p.fetchPage(fetchStatusAt, fetchScheduledAt)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		p.clear()
	}
	return items, nil
}

func (p *mongoExecutionBatchPager) fetchPage(
	fetchStatusAt func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error),
	fetchScheduledAt func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error),
) ([]testkube.TestWorkflowExecution, error) {
	items := make([]testkube.TestWorkflowExecution, 0, executionUpdatesBatchSize)
	for len(items) < executionUpdatesBatchSize {
		if err := p.fillBuffer(&p.statusAtBuffer, &p.statusAtExhausted, p.statusAtCursor, fetchStatusAt); err != nil {
			return nil, err
		}
		if err := p.fillBuffer(&p.scheduledAtBuffer, &p.scheduledExhausted, p.scheduledAtCursor, fetchScheduledAt); err != nil {
			return nil, err
		}

		next, source := p.nextExecution()
		if source == "" {
			break
		}
		items = append(items, next)
		p.advance(source)
	}

	if len(items) == 0 {
		return nil, nil
	}
	return items, nil
}

func (p *mongoExecutionBatchPager) fillBuffer(
	buffer *[]testkube.TestWorkflowExecution,
	exhausted *bool,
	after *executionBatchCursor,
	fetch func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error),
) error {
	if *exhausted || len(*buffer) > 0 {
		return nil
	}

	items, err := fetch(p.snapshotBefore, after)
	if err != nil {
		return err
	}
	*buffer = items
	if len(items) < executionUpdatesBatchSize {
		*exhausted = true
	}
	return nil
}

func (p *mongoExecutionBatchPager) nextExecution() (testkube.TestWorkflowExecution, string) {
	switch {
	case len(p.statusAtBuffer) == 0 && len(p.scheduledAtBuffer) == 0:
		return testkube.TestWorkflowExecution{}, ""
	case len(p.scheduledAtBuffer) == 0:
		return p.statusAtBuffer[0], "status"
	case len(p.statusAtBuffer) == 0:
		return p.scheduledAtBuffer[0], "scheduled"
	case pendingExecutionLess(p.statusAtBuffer[0], p.scheduledAtBuffer[0]):
		return p.statusAtBuffer[0], "status"
	default:
		return p.scheduledAtBuffer[0], "scheduled"
	}
}

func (p *mongoExecutionBatchPager) advance(source string) {
	switch source {
	case "status":
		p.statusAtCursor = executionCursorOf(p.statusAtBuffer[0])
		p.statusAtBuffer = p.statusAtBuffer[1:]
	case "scheduled":
		p.scheduledAtCursor = executionCursorOf(p.scheduledAtBuffer[0])
		p.scheduledAtBuffer = p.scheduledAtBuffer[1:]
	}
}

func (p *mongoExecutionBatchPager) active() bool {
	return !p.snapshotBefore.IsZero()
}

func (p *mongoExecutionBatchPager) hasProgress() bool {
	return p.statusAtCursor != nil ||
		p.scheduledAtCursor != nil ||
		len(p.statusAtBuffer) > 0 ||
		len(p.scheduledAtBuffer) > 0
}

func (p *mongoExecutionBatchPager) resetSnapshot() {
	p.snapshotBefore = p.currentTime()
	p.clearProgress()
}

func (p *mongoExecutionBatchPager) clear() {
	p.snapshotBefore = time.Time{}
	p.clearProgress()
}

func (p *mongoExecutionBatchPager) clearProgress() {
	p.statusAtCursor = nil
	p.scheduledAtCursor = nil
	p.statusAtBuffer = nil
	p.scheduledAtBuffer = nil
	p.statusAtExhausted = false
	p.scheduledExhausted = false
}

func (p *mongoExecutionBatchPager) currentTime() time.Time {
	if p.now != nil {
		return p.now().UTC()
	}
	return time.Now().UTC()
}

func executionCursorOf(exe testkube.TestWorkflowExecution) *executionBatchCursor {
	return &executionBatchCursor{
		pendingAt:   pendingExecutionTime(exe),
		executionID: exe.Id,
	}
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
