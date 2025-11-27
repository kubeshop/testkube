package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type stubRunner struct {
	executeFn func(executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error)
}

func (s *stubRunner) Execute(req executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	if s.executeFn != nil {
		return s.executeFn(req)
	}
	return &executionworkertypes.ExecuteResult{}, nil
}

func (s *stubRunner) Monitor(context.Context, string, string, string) error { return nil }

func (s *stubRunner) Notifications(context.Context, string) executionworkertypes.NotificationsWatcher {
	return nil
}

func (s *stubRunner) Pause(string) error  { return nil }
func (s *stubRunner) Resume(string) error { return nil }
func (s *stubRunner) Abort(string) error  { return nil }
func (s *stubRunner) Cancel(string) error { return nil }

func TestDirectRunTestWorkflow_EmitsStartEventInNewArchitecture(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	client := controlplaneclient.NewMockClient(ctrl)
	status := testkube.QUEUED_TestWorkflowStatus
	execution := &testkube.TestWorkflowExecution{
		Id:               "exec-1",
		Name:             "workflow-1",
		GroupId:          "group-1",
		Number:           1,
		Result:           &testkube.TestWorkflowResult{Status: &status},
		ResolvedWorkflow: &testkube.TestWorkflow{Name: "workflow-1"},
	}
	client.EXPECT().GetExecution(gomock.Any(), "env-1", "exec-1").Return(execution, nil)
	client.EXPECT().InitExecution(gomock.Any(), "env-1", "exec-1", gomock.Any(), gomock.Any()).Return(nil)

	emitter := &eventRecorder{}
	result := &executionworkertypes.ExecuteResult{
		Namespace: "ns-1",
		Signature: []testkube.TestWorkflowSignature{},
		Redundant: false,
	}
	runner := &stubRunner{executeFn: func(executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
		return result, nil
	}}

	proCtx := config.ProContext{
		OrgSlug: "org-slug",
		EnvID:   "env-1",
		EnvSlug: "env-slug",
		Agent: config.ProContextAgent{
			ID:           "agent-1",
			Environments: []config.ProContextAgentEnvironment{{ID: "env-1", Slug: "env-slug"}},
		},
	}

	loop := &agentLoop{
		runner:             runner,
		worker:             nil,
		logger:             zap.NewNop().Sugar(),
		emitter:            emitter,
		client:             client,
		proContext:         proCtx,
		controlPlaneConfig: testworkflowconfig.ControlPlaneConfig{},
		organizationId:     "org-id",
	}

	require.NoError(t, loop.directRunTestWorkflow("env-1", "exec-1", "token", nil))

	events := emitter.drain()
	require.NotEmpty(t, events)
	require.Equal(t, *testkube.EventStartTestWorkflow, events[len(events)-1].Type())
}
