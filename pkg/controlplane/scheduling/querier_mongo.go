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

type mongoExecutionBatchPagerState struct {
	statusAtCursor     *executionBatchCursor
	scheduledAtCursor  *executionBatchCursor
	statusAtBuffer     []testkube.TestWorkflowExecution
	scheduledAtBuffer  []testkube.TestWorkflowExecution
	statusAtOffset     int
	scheduledAtOffset  int
	statusAtExhausted  bool
	scheduledExhausted bool
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
	base := p.state()
	state := base
	items := make([]testkube.TestWorkflowExecution, 0, executionUpdatesBatchSize)
	for len(items) < executionUpdatesBatchSize {
		if err := state.fillBuffer(p.snapshotBefore, &state.statusAtBuffer, &state.statusAtExhausted, state.statusAtCursor, fetchStatusAt); err != nil {
			p.restoreFetchedState(base, state)
			return nil, err
		}
		if err := state.fillBuffer(p.snapshotBefore, &state.scheduledAtBuffer, &state.scheduledExhausted, state.scheduledAtCursor, fetchScheduledAt); err != nil {
			p.restoreFetchedState(base, state)
			return nil, err
		}

		next, source := state.nextExecution()
		if source == "" {
			break
		}
		items = append(items, next)
		state.advance(source)
	}

	if len(items) == 0 {
		p.restoreState(state)
		return nil, nil
	}
	p.restoreState(state)
	return items, nil
}

func (s *mongoExecutionBatchPagerState) fillBuffer(
	snapshotBefore time.Time,
	buffer *[]testkube.TestWorkflowExecution,
	exhausted *bool,
	after *executionBatchCursor,
	fetch func(snapshotBefore time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error),
) error {
	if *exhausted || s.bufferHasItems(buffer) {
		return nil
	}

	items, err := fetch(snapshotBefore, after)
	if err != nil {
		return err
	}
	*buffer = append(*buffer, items...)
	if len(items) < executionUpdatesBatchSize {
		*exhausted = true
	}
	return nil
}

func (s *mongoExecutionBatchPagerState) nextExecution() (testkube.TestWorkflowExecution, string) {
	switch {
	case s.statusAtOffset >= len(s.statusAtBuffer) && s.scheduledAtOffset >= len(s.scheduledAtBuffer):
		return testkube.TestWorkflowExecution{}, ""
	case s.scheduledAtOffset >= len(s.scheduledAtBuffer):
		return s.statusAtBuffer[s.statusAtOffset], "status"
	case s.statusAtOffset >= len(s.statusAtBuffer):
		return s.scheduledAtBuffer[s.scheduledAtOffset], "scheduled"
	case pendingExecutionLess(s.statusAtBuffer[s.statusAtOffset], s.scheduledAtBuffer[s.scheduledAtOffset]):
		return s.statusAtBuffer[s.statusAtOffset], "status"
	default:
		return s.scheduledAtBuffer[s.scheduledAtOffset], "scheduled"
	}
}

func (s *mongoExecutionBatchPagerState) advance(source string) {
	switch source {
	case "status":
		s.statusAtCursor = executionCursorOf(s.statusAtBuffer[s.statusAtOffset])
		s.statusAtOffset++
	case "scheduled":
		s.scheduledAtCursor = executionCursorOf(s.scheduledAtBuffer[s.scheduledAtOffset])
		s.scheduledAtOffset++
	}
}

func (s *mongoExecutionBatchPagerState) bufferHasItems(buffer *[]testkube.TestWorkflowExecution) bool {
	switch {
	case buffer == &s.statusAtBuffer:
		return s.statusAtOffset < len(s.statusAtBuffer)
	case buffer == &s.scheduledAtBuffer:
		return s.scheduledAtOffset < len(s.scheduledAtBuffer)
	default:
		return len(*buffer) > 0
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

func (p *mongoExecutionBatchPager) state() mongoExecutionBatchPagerState {
	return mongoExecutionBatchPagerState{
		statusAtCursor:     cloneExecutionBatchCursor(p.statusAtCursor),
		scheduledAtCursor:  cloneExecutionBatchCursor(p.scheduledAtCursor),
		statusAtBuffer:     append([]testkube.TestWorkflowExecution(nil), p.statusAtBuffer...),
		scheduledAtBuffer:  append([]testkube.TestWorkflowExecution(nil), p.scheduledAtBuffer...),
		statusAtExhausted:  p.statusAtExhausted,
		scheduledExhausted: p.scheduledExhausted,
	}
}

func (p *mongoExecutionBatchPager) restoreState(state mongoExecutionBatchPagerState) {
	p.statusAtCursor = cloneExecutionBatchCursor(state.statusAtCursor)
	p.scheduledAtCursor = cloneExecutionBatchCursor(state.scheduledAtCursor)
	p.statusAtBuffer = append([]testkube.TestWorkflowExecution(nil), state.statusAtBuffer[state.statusAtOffset:]...)
	p.scheduledAtBuffer = append([]testkube.TestWorkflowExecution(nil), state.scheduledAtBuffer[state.scheduledAtOffset:]...)
	p.statusAtExhausted = state.statusAtExhausted
	p.scheduledExhausted = state.scheduledExhausted
}

func (p *mongoExecutionBatchPager) restoreFetchedState(base, working mongoExecutionBatchPagerState) {
	p.statusAtCursor = cloneExecutionBatchCursor(base.statusAtCursor)
	p.scheduledAtCursor = cloneExecutionBatchCursor(base.scheduledAtCursor)
	p.statusAtBuffer = append([]testkube.TestWorkflowExecution(nil), working.statusAtBuffer...)
	p.scheduledAtBuffer = append([]testkube.TestWorkflowExecution(nil), working.scheduledAtBuffer...)
	p.statusAtExhausted = working.statusAtExhausted
	p.scheduledExhausted = working.scheduledExhausted
}

func cloneExecutionBatchCursor(cursor *executionBatchCursor) *executionBatchCursor {
	if cursor == nil {
		return nil
	}
	cloned := *cursor
	return &cloned
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
