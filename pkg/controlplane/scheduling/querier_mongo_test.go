package scheduling

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestPendingExecutionFilterUsesNativePendingFields(t *testing.T) {
	snapshotBefore := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	after := &executionBatchCursor{pendingAt: snapshotBefore.Add(-time.Minute), executionID: "exec-2"}

	withStatusAt := pendingExecutionFilter([]testkube.TestWorkflowStatus{testkube.STARTING_TestWorkflowStatus}, snapshotBefore, after, "statusat", true)
	withScheduledAt := pendingExecutionFilter([]testkube.TestWorkflowStatus{testkube.ASSIGNED_TestWorkflowStatus}, snapshotBefore, after, "scheduledat", false)

	require.Equal(t, bson.M{"statusat": bson.M{"$lte": snapshotBefore}}, withStatusAt["$and"].(bson.A)[1])
	require.Equal(t, bson.M{"$gt": time.Time{}}, withStatusAt["$and"].(bson.A)[2].(bson.M)["statusat"])
	require.Equal(t, bson.M{"scheduledat": bson.M{"$lte": snapshotBefore}}, withScheduledAt["$and"].(bson.A)[1])
	require.Equal(t, bson.M{"$gt": after.pendingAt}, withScheduledAt["$and"].(bson.A)[3].(bson.M)["$or"].(bson.A)[0].(bson.M)["scheduledat"])
}

func TestMongoExecutionBatchPagerOrdersByPendingTransitionTime(t *testing.T) {
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	statusExecutions := []testkube.TestWorkflowExecution{
		{Id: "002", StatusAt: base.Add(2 * time.Second)},
		{Id: "004", StatusAt: base.Add(4 * time.Second)},
	}
	scheduledExecutions := []testkube.TestWorkflowExecution{
		{Id: "001", ScheduledAt: base.Add(1 * time.Second)},
		{Id: "003", ScheduledAt: base.Add(3 * time.Second)},
	}
	pager := mongoExecutionBatchPager{
		now: func() time.Time { return base },
	}

	page, err := pager.Next(
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(statusExecutions, after, executionUpdatesBatchSize), nil
		},
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(scheduledExecutions, after, executionUpdatesBatchSize), nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"001", "002", "003", "004"}, executionIDs(page))
}

func TestMongoExecutionBatchPagerCarriesOverRowsAcrossPages(t *testing.T) {
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	statusExecutions := make([]testkube.TestWorkflowExecution, 0, executionUpdatesBatchSize)
	scheduledExecutions := []testkube.TestWorkflowExecution{
		{Id: "001", ScheduledAt: base},
	}

	for i := 0; i < executionUpdatesBatchSize; i++ {
		statusExecutions = append(statusExecutions, testkube.TestWorkflowExecution{
			Id:       fmt.Sprintf("%03d", i+2),
			StatusAt: base,
		})
	}

	pager := mongoExecutionBatchPager{
		now: func() time.Time { return base },
	}

	page1, err := pager.Next(
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(statusExecutions, after, executionUpdatesBatchSize), nil
		},
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(scheduledExecutions, after, executionUpdatesBatchSize), nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, executionUpdatesBatchSize, len(page1))
	require.Equal(t, "001", page1[0].Id)
	require.Equal(t, "100", page1[len(page1)-1].Id)

	page2, err := pager.Next(
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(statusExecutions, after, executionUpdatesBatchSize), nil
		},
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(scheduledExecutions, after, executionUpdatesBatchSize), nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"101"}, executionIDs(page2))
}

func TestMongoExecutionBatchPagerStartsFreshSnapshotAfterIdlePoll(t *testing.T) {
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	now := base
	statusExecutions := []testkube.TestWorkflowExecution(nil)
	scheduledExecutions := []testkube.TestWorkflowExecution(nil)

	pager := mongoExecutionBatchPager{
		now: func() time.Time { return now },
	}

	page, err := pager.Next(
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(statusExecutions, after, executionUpdatesBatchSize), nil
		},
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(scheduledExecutions, after, executionUpdatesBatchSize), nil
		},
	)
	require.NoError(t, err)
	require.Nil(t, page)

	now = base.Add(time.Second)
	scheduledExecutions = append(scheduledExecutions, testkube.TestWorkflowExecution{
		Id:          "001",
		ScheduledAt: now,
	})

	page, err = pager.Next(
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(statusExecutions, after, executionUpdatesBatchSize), nil
		},
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(scheduledExecutions, after, executionUpdatesBatchSize), nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"001"}, executionIDs(page))
}

func TestMongoExecutionBatchPagerStartsFreshSnapshotAfterDrainAndIdlePoll(t *testing.T) {
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	now := base
	statusExecutions := []testkube.TestWorkflowExecution{
		{Id: "001", StatusAt: base},
	}
	scheduledExecutions := []testkube.TestWorkflowExecution(nil)

	pager := mongoExecutionBatchPager{
		now: func() time.Time { return now },
	}

	page, err := pager.Next(
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(statusExecutions, after, executionUpdatesBatchSize), nil
		},
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(scheduledExecutions, after, executionUpdatesBatchSize), nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"001"}, executionIDs(page))

	statusExecutions = nil
	page, err = pager.Next(
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(statusExecutions, after, executionUpdatesBatchSize), nil
		},
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(scheduledExecutions, after, executionUpdatesBatchSize), nil
		},
	)
	require.NoError(t, err)
	require.Nil(t, page)

	now = base.Add(time.Second)
	scheduledExecutions = append(scheduledExecutions, testkube.TestWorkflowExecution{
		Id:          "002",
		ScheduledAt: now,
	})

	page, err = pager.Next(
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(statusExecutions, after, executionUpdatesBatchSize), nil
		},
		func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
			return filterAfter(scheduledExecutions, after, executionUpdatesBatchSize), nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"002"}, executionIDs(page))
}

func filterAfter(executions []testkube.TestWorkflowExecution, after *executionBatchCursor, limit int) []testkube.TestWorkflowExecution {
	filtered := make([]testkube.TestWorkflowExecution, 0, limit)
	for _, exe := range executions {
		if after != nil && !pendingExecutionLess(testkube.TestWorkflowExecution{
			Id:          after.executionID,
			StatusAt:    after.pendingAt,
			ScheduledAt: after.pendingAt,
		}, exe) {
			continue
		}
		filtered = append(filtered, exe)
		if len(filtered) == limit {
			break
		}
	}
	return filtered
}

func executionIDs(executions []testkube.TestWorkflowExecution) []string {
	ids := make([]string, 0, len(executions))
	for _, exe := range executions {
		ids = append(ids, exe.Id)
	}
	return ids
}
