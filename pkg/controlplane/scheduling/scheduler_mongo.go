package scheduling

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/utils"
)

type MongoScheduler struct {
	executionsCollection *mongo.Collection
}

func NewMongoScheduler(col *mongo.Collection) Scheduler {
	return &MongoScheduler{executionsCollection: col}
}

func (s *MongoScheduler) ScheduleExecution(ctx context.Context, info RunnerInfo) (execution testkube.TestWorkflowExecution, found bool, e error) {
	// Note: Standalone Control Plane does not support policies.
	// Note: Standalone Control Plane does not support label matches, excludes, etc. It always targets the DefaultRunner.

	filter := bson.M{"$and": bson.A{
		bson.M{"result.status": bson.M{"$in": bson.A{
			testkube.QUEUED_TestWorkflowStatus,
			testkube.ASSIGNED_TestWorkflowStatus,
			testkube.STARTING_TestWorkflowStatus,
			"", nil, // Both of these combined count as "nothing".
		}}},
		bson.M{"$or": bson.A{
			bson.M{"runnerid": info.Id},
			bson.M{"runnerid": bson.M{"$in": bson.A{"", nil}}},
		}},
	}}

	now := time.Now()
	update := bson.A{ // Update must be a pipeline so that we can use a conditional for the status update.
		// Fill in any missing status values with an empty string so that they can be compared as nil comparison
		// in a $cond seems to be a difficult thing to achieve.
		bson.M{"$fill": bson.M{
			"output": bson.M{"result.status": bson.M{"value": ""}},
		}},
		bson.M{"$set": bson.M{
			// Only modify the assigned time if we are actually assigning this to a new runner.
			"assignedat": bson.M{"$cond": bson.M{
				"if":   bson.M{"$ne": bson.A{"$runnerid", info.Id}},
				"then": now,
				"else": "$assignedat",
			}},
			// Only modify the status change timestamp if we are about to modify the status of the
			// execution. Otherwise leave it alone as the state is not being transitioned. For example
			// a STARTING execution that is being retrieved again to be retried.
			"statusat": bson.M{"$cond": bson.M{
				"if": bson.M{"$in": bson.A{"$result.status", bson.A{
					testkube.QUEUED_TestWorkflowStatus,
					"", // Should at least be an empty string here because of previous fill.
				}}},
				"then": now,
				"else": "$statusat",
			}},
			// Only modify the status to PENDING if it was QUEUED or nothing. Otherwise leave it
			// alone as the state is not being transitioned. For example a STARTING execution that
			// is being retrieved again to be retried.
			"result.status": bson.M{"$cond": bson.M{
				"if": bson.M{"$in": bson.A{"$result.status", bson.A{
					testkube.QUEUED_TestWorkflowStatus,
					"", // Should at least be an empty string here because of previous fill.
				}}},
				"then": testkube.ASSIGNED_TestWorkflowStatus,
				"else": "$result.status",
			}},
		}},
		// Set runner id last to avoid conflicts with conditional checks.
		bson.M{"$set": bson.M{
			"runnerid": info.Id,
		}},
	}

	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After). // Always return the updated document.
		SetSort(bson.M{"scheduledat": 1}) // Choose the oldest scheduled match first.

	result := s.executionsCollection.FindOneAndUpdate(ctx, filter, update, opts)
	switch err := result.Err(); {
	case utils.IsNotFound(err):
		return testkube.TestWorkflowExecution{}, false, nil
	case err != nil:
		return testkube.TestWorkflowExecution{}, false, fmt.Errorf("find one and update: %w", err)
	}

	var ret testkube.TestWorkflowExecution
	if err := result.Decode(&ret); err != nil {
		return testkube.TestWorkflowExecution{}, false, fmt.Errorf("decode returned execution: %w", err)
	}

	return ret, true, nil
}
