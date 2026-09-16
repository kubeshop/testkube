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
