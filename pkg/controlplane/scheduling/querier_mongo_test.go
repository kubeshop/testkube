package scheduling

import (
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

func TestMergePendingExecutionsOrdersByPendingTransitionTime(t *testing.T) {
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	withStatusAt := []testkube.TestWorkflowExecution{
		{Id: "exec-2", StatusAt: base.Add(2 * time.Second)},
		{Id: "exec-4", StatusAt: base.Add(4 * time.Second)},
	}
	withScheduledAt := []testkube.TestWorkflowExecution{
		{Id: "exec-1", ScheduledAt: base.Add(1 * time.Second)},
		{Id: "exec-3", ScheduledAt: base.Add(3 * time.Second)},
	}

	executions := mergePendingExecutions(withStatusAt, withScheduledAt, 3)

	require.Equal(t, []string{"exec-1", "exec-2", "exec-3"}, []string{executions[0].Id, executions[1].Id, executions[2].Id})
}

func TestMergePendingExecutionsCarriesOverRowsAcrossPages(t *testing.T) {
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	statusExecutions := []testkube.TestWorkflowExecution{
		{Id: "002", StatusAt: base},
		{Id: "004", StatusAt: base},
		{Id: "006", StatusAt: base},
		{Id: "008", StatusAt: base},
	}
	scheduledExecutions := []testkube.TestWorkflowExecution{
		{Id: "001", ScheduledAt: base},
		{Id: "003", ScheduledAt: base},
		{Id: "005", ScheduledAt: base},
		{Id: "007", ScheduledAt: base},
	}
	pager := executionBatchPager[testkube.TestWorkflowExecution]{
		now: func() time.Time { return base },
	}

	fetch := func(_ time.Time, after *executionBatchCursor) ([]testkube.TestWorkflowExecution, error) {
		return mergePendingExecutions(
			filterAfter(statusExecutions, after, 3),
			filterAfter(scheduledExecutions, after, 3),
			3,
		), nil
	}
	cursorOf := func(exe testkube.TestWorkflowExecution) *executionBatchCursor {
		return &executionBatchCursor{pendingAt: pendingExecutionTime(exe), executionID: exe.Id}
	}

	page1, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []string{"001", "002", "003"}, executionIDs(page1))

	page2, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []string{"004", "005", "006"}, executionIDs(page2))

	page3, err := pager.Next(fetch, cursorOf)
	require.NoError(t, err)
	require.Equal(t, []string{"007", "008"}, executionIDs(page3))
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
