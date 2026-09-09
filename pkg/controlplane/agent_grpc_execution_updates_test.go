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

// The scheduler records the rerun policy on the execution, so this path has to
// read it back. Dropping it does not fail anything: the pod receives no
// selection and runs the whole suite, so a rerun asked to re-run forty test
// cases reports success having re-run ten thousand.
func TestCreateExecutionStart_CarriesTheRerunPolicy(t *testing.T) {
	start := createExecutionStart(testkube.TestWorkflowExecution{
		Id: "exec-2",
		Rerun: &testkube.TestWorkflowRerun{
			ExecutionId: "exec-1",
			OnlyFailed:  true,
			TestCases:   []string{"tests.a::test_one", "tests.b::test_two"},
		},
	}, scheduling.RunnerInfo{EnvironmentId: "env-1"})

	rerun := start.GetRerun()
	require.NotNil(t, rerun)
	require.Equal(t, "exec-1", rerun.GetExecutionId())
	require.True(t, rerun.GetOnlyFailed())
	require.Equal(t, []string{"tests.a::test_one", "tests.b::test_two"}, rerun.GetTestCases())
}

// Nil has to stay nil: the pod reads a non-nil policy as "this run was
// narrowed", and almost no execution is a rerun.
func TestCreateExecutionStart_LeavesRerunUnsetWithoutAPolicy(t *testing.T) {
	start := createExecutionStart(testkube.TestWorkflowExecution{Id: "exec-1"}, scheduling.RunnerInfo{})

	require.Nil(t, start.GetRerun())
}

// The record's slice is not ours to hand out - the execution outlives this call.
func TestCreateExecutionStart_ClonesTheTestCases(t *testing.T) {
	exe := testkube.TestWorkflowExecution{
		Id:    "exec-2",
		Rerun: &testkube.TestWorkflowRerun{TestCases: []string{"tests.a::test_one"}},
	}

	start := createExecutionStart(exe, scheduling.RunnerInfo{})
	start.GetRerun().TestCases[0] = "mutated"

	require.Equal(t, []string{"tests.a::test_one"}, exe.Rerun.TestCases)
}
