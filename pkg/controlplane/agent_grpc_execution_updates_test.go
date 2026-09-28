package controlplane

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
)

func TestCreateExecutionStart_PropagatesTags(t *testing.T) {
	exe := testkube.TestWorkflowExecution{
		Id:              "exec-1",
		GroupId:         "group-1",
		Name:            "workflow-1-1",
		Number:          1,
		ScheduledAt:     time.Unix(1735689600, 0),
		DisableWebhooks: true,
		Tags:            map[string]string{"env": "prod", "suite": "smoke"},
		RunningContext:  &testkube.TestWorkflowRunningContext{Actor: &testkube.TestWorkflowRunningContextActor{ExecutionPath: "p1/p2"}},
		Workflow:        &testkube.TestWorkflow{Name: "wf-a"},
	}
	info := scheduling.RunnerInfo{EnvironmentId: "env-1"}

	start := createExecutionStart(exe, info)

	require.Equal(t, exe.Tags, start.GetTags())
	require.Equal(t, "env-1", start.GetEnvironmentId())
	require.Equal(t, []string{"p1", "p2"}, start.GetAncestorExecutionIds())
	require.Equal(t, "wf-a", start.GetWorkflowName())
}

// The scheduler records lineage on the execution rather than handing it to the
// runner, so every ExecutionStart writer has to read it back off the record.
// Omitting it does not fail: the pod simply cannot resolve execution("rerun"),
// and a workflow written against its own previous run loses the reference with
// nothing to say so.
func TestCreateExecutionStart_CarriesLineage(t *testing.T) {
	start := createExecutionStart(testkube.TestWorkflowExecution{
		Id:      "exec-3",
		Lineage: &testkube.TestWorkflowExecutionLineage{BaseId: "exec-2", RootId: "exec-1", Attempt: 3},
	}, scheduling.RunnerInfo{EnvironmentId: "env-1"})

	lineage := start.GetLineage()
	require.NotNil(t, lineage)
	require.Equal(t, "exec-2", lineage.GetBaseExecutionId())
	require.Equal(t, "exec-1", lineage.GetRootExecutionId())
	require.Equal(t, int32(3), lineage.GetAttempt())
}

// An execution recorded before lineage existed carries none, and nil stays nil
// so the pod reads no rerun rather than a zeroed record.
func TestCreateExecutionStart_LeavesLineageUnsetWithoutARecord(t *testing.T) {
	start := createExecutionStart(testkube.TestWorkflowExecution{Id: "exec-1"}, scheduling.RunnerInfo{})

	require.Nil(t, start.GetLineage())
}
