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

// MongoExecutionQuerier reads the work the runner has to be told about from the
// test workflow execution collection.
type MongoExecutionQuerier struct {
	executionsCollection *mongo.Collection
}

func NewMongoExecutionQuerier(col *mongo.Collection) *MongoExecutionQuerier {
	return &MongoExecutionQuerier{executionsCollection: col}
}

// dispatchProjection is the set of fields the runner is sent. Postgres gets this
// for free by reading one table; mongo stores the whole execution in a single
// document, so it has to ask, or every poll drags back the full workflow spec of
// every queued execution.
var dispatchProjection = bson.M{
	"id":                     1,
	"groupid":                1,
	"name":                   1,
	"number":                 1,
	"scheduledat":            1,
	"assignedat":             1,
	"statusat":               1,
	"disablewebhooks":        1,
	"tags":                   1,
	"runningcontext":         1,
	"runtime":                1,
	"lineage":                1,
	"silentmode":             1,
	"result.status":          1,
	"result.predictedstatus": 1,
	"workflow.name":          1,
}

// Transitions returns every execution awaiting a pause, resume or stop.
func (a MongoExecutionQuerier) Transitions(ctx context.Context) ([]ExecutionTransition, error) {
	filter := bson.M{"result.status": bson.M{"$in": bson.A{
		testkube.PAUSING_TestWorkflowStatus,
		testkube.RESUMING_TestWorkflowStatus,
		testkube.STOPPING_TestWorkflowStatus,
	}}}
	opts := options.Find().SetProjection(bson.M{
		"id":                     1,
		"result.status":          1,
		"result.predictedstatus": 1,
	})

	executions, err := a.find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find executions awaiting a transition: %w", err)
	}

	transitions := make([]ExecutionTransition, 0, len(executions))
	for _, exe := range executions {
		if exe.Result == nil || exe.Result.Status == nil {
			continue
		}
		var predicted testkube.TestWorkflowStatus
		if exe.Result.PredictedStatus != nil {
			predicted = *exe.Result.PredictedStatus
		}
		transitions = append(transitions, ExecutionTransition{
			Id:              exe.Id,
			Status:          *exe.Result.Status,
			PredictedStatus: predicted,
		})
	}
	return transitions, nil
}

// ToStart returns at most limit executions to dispatch, oldest scheduled first.
func (a MongoExecutionQuerier) ToStart(ctx context.Context, limit int, redispatchBefore time.Time) ([]testkube.TestWorkflowExecution, error) {
	filter := bson.M{"$or": bson.A{
		bson.M{"result.status": testkube.ASSIGNED_TestWorkflowStatus},
		bson.M{
			"result.status": testkube.STARTING_TestWorkflowStatus,
			"statusat":      bson.M{"$lt": redispatchBefore},
		},
	}}
	// The sort is not cosmetic: without it mongo dispatched in natural order,
	// which is the opposite of the postgres path and starves the oldest work.
	opts := options.Find().
		SetSort(bson.D{{Key: "scheduledat", Value: 1}}).
		SetLimit(int64(limit)).
		SetProjection(dispatchProjection)

	executions, err := a.find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find executions to start: %w", err)
	}
	return executions, nil
}

// StaleStarting returns the ids of executions the runner never acknowledged.
func (a MongoExecutionQuerier) StaleStarting(ctx context.Context, staleBefore time.Time, limit int) ([]string, error) {
	filter := bson.M{
		"result.status": testkube.STARTING_TestWorkflowStatus,
		"statusat":      bson.M{"$lt": staleBefore},
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "scheduledat", Value: 1}}).
		SetLimit(int64(limit)).
		SetProjection(bson.M{"id": 1})

	executions, err := a.find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find stale starting executions: %w", err)
	}

	ids := make([]string, 0, len(executions))
	for _, exe := range executions {
		ids = append(ids, exe.Id)
	}
	return ids, nil
}

// find drains the cursor into a slice.
//
// A decode failure on one document skips that document rather than abandoning
// the rest: one malformed record must not stop the runner being told about
// everything behind it.
func (a MongoExecutionQuerier) find(ctx context.Context, filter any, opts *options.FindOptionsBuilder) ([]testkube.TestWorkflowExecution, error) {
	cur, err := a.executionsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var executions []testkube.TestWorkflowExecution
	for cur.Next(ctx) {
		var exe testkube.TestWorkflowExecution
		if err := cur.Decode(&exe); err != nil {
			continue
		}
		executions = append(executions, exe)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return executions, nil
}
