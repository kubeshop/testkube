package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
)

func TestExecutionConfigFromStart_PropagatesTags(t *testing.T) {
	queuedAt := time.Unix(1735689600, 0)
	start := &executionv1.ExecutionStart{
		ExecutionId:          ptr("exec-1"),
		GroupId:              ptr("group-1"),
		Name:                 ptr("wf-1-1"),
		Number:               ptr(int32(1)),
		QueuedAt:             timestamppb.New(queuedAt),
		DisableWebhooks:      ptr(true),
		EnvironmentId:        ptr("env-1"),
		AncestorExecutionIds: []string{"a", "b"},
		Tags:                 map[string]string{"env": "staging", "suite": "smoke"},
	}

	cfg := executionConfigFromStart(start, "org-1", nil)

	require.Equal(t, "exec-1", cfg.Id)
	require.Equal(t, "org-1", cfg.OrganizationId)
	require.Equal(t, "a/b", cfg.ParentIds)
	require.Equal(t, queuedAt.Unix(), cfg.ScheduledAt.Unix())
	require.Equal(t, map[string]string{"env": "staging", "suite": "smoke"}, cfg.Tags)

	start.Tags["env"] = "mutated"
	require.Equal(t, "staging", cfg.Tags["env"])
}

func ptr[T any](v T) *T {
	return &v
}

// The control plane records lineage on the execution and projects it onto the
// ExecutionStart, so both halves of this hop have to carry it. Either half
// dropping it leaves the pod unable to resolve execution("rerun") on a rerun,
// with no error to say so - the paired writer is lineageProtoOf in
// pkg/controlplane/agent_grpc_execution_updates.go.
func TestExecutionConfigFromStart_CarriesLineage(t *testing.T) {
	start := &executionv1.ExecutionStart{
		ExecutionId: ptr("exec-3"),
		Lineage: &executionv1.ExecutionLineage{
			BaseExecutionId: ptr("exec-2"),
			RootExecutionId: ptr("exec-1"),
			Attempt:         ptr(int32(3)),
		},
	}

	cfg := executionConfigFromStart(start, "org-1", nil)

	require.NotNil(t, cfg.Lineage)
	require.Equal(t, "exec-2", cfg.Lineage.BaseId)
	require.Equal(t, "exec-1", cfg.Lineage.RootId)
	require.Equal(t, int32(3), cfg.Lineage.Attempt)
}

// An ordinary execution carries none, and nil has to stay nil.
func TestExecutionConfigFromStart_NoLineage(t *testing.T) {
	cfg := executionConfigFromStart(&executionv1.ExecutionStart{ExecutionId: ptr("exec-1")}, "org-1", nil)
	require.Nil(t, cfg.Lineage)
}

// alwaysFailingExecutionService answers every poll with an error. The embedded interface is nil,
// so a call to any other method panics and names itself rather than returning a wrong answer.
type alwaysFailingExecutionService struct {
	executionv1.TestWorkflowExecutionServiceClient
}

func (alwaysFailingExecutionService) GetExecutionUpdates(context.Context, *executionv1.GetExecutionUpdatesRequest, ...grpc.CallOption) (*executionv1.GetExecutionUpdatesResponse, error) {
	return nil, status.Error(codes.DeadlineExceeded, "context deadline exceeded")
}

// pollUntilWaits runs the poll loop against a control plane that never answers, and returns the
// waits it asked for and the waits it logged. It replaces the wait itself, because the point is
// the durations the loop chooses and not the time they would take.
func pollUntilWaits(t *testing.T, count int) (asked []time.Duration, logged []time.Duration) {
	t.Helper()

	core, recorded := observer.New(zapcore.WarnLevel)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	c := Client{
		client:       alwaysFailingExecutionService{},
		logger:       zap.New(core).Sugar(),
		callTimeout:  defaultCallTimeout,
		pollInterval: time.Millisecond,
	}
	c.sleep = func(_ context.Context, d time.Duration) error {
		asked = append(asked, d)
		if len(asked) >= count {
			cancel()
			return context.Canceled
		}
		return nil
	}

	require.Error(t, c.Start(ctx, "env-1"))
	require.Len(t, asked, count)

	for _, entry := range recorded.All() {
		d, ok := entry.ContextMap()["backoff"].(time.Duration)
		require.True(t, ok, "every backoff log must carry the duration")
		logged = append(logged, d)
	}
	return asked, logged
}

func TestClient_Start_CapsTheWaitAfterConsecutiveFailures(t *testing.T) {
	asked, _ := pollUntilWaits(t, 25)

	for i, d := range asked {
		assert.LessOrEqual(t, d, maxPollBackoff, "wait %d of a run of failures exceeded the cap", i+1)
	}
	assert.Greater(t, asked[len(asked)-1], time.Duration(0), "the wait must still grow with failures")
}

func TestClient_Start_LogsTheWaitItTakes(t *testing.T) {
	asked, logged := pollUntilWaits(t, 10)

	assert.Equal(t, asked, logged,
		"the logged wait must be the wait taken, or the attempt counter advances twice per failure")
}

func TestClient_waitBackoff_ReturnsWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := Client{}.waitBackoff(ctx, time.Hour)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), time.Second, "a shutdown must not sit out the remaining wait")
}
