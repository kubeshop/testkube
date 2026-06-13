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
		Id:               "exec-1",
		GroupId:          "group-1",
		Name:             "workflow-1-1",
		Number:           1,
		ScheduledAt:      time.Unix(1735689600, 0),
		DisableWebhooks:  true,
		Tags:             map[string]string{"env": "prod", "suite": "smoke"},
		RunningContext:   &testkube.TestWorkflowRunningContext{Actor: &testkube.TestWorkflowRunningContextActor{ExecutionPath: "p1/p2"}},
		Workflow:         &testkube.TestWorkflow{Name: "wf-a"},
	}
	info := scheduling.RunnerInfo{EnvironmentId: "env-1"}

	start := createExecutionStart(exe, info)

	require.Equal(t, exe.Tags, start.GetTags())
	require.Equal(t, "env-1", start.GetEnvironmentId())
	require.Equal(t, []string{"p1", "p2"}, start.GetAncestorExecutionIds())
	require.Equal(t, "wf-a", start.GetWorkflowName())
}
