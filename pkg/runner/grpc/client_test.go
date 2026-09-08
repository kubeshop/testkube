package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

// TestRunningContextFromProto_CarriesExecutionReference covers a gap that
// predates the rerun work: the message did not declare the field, so this start
// path could not carry it, and execution.runningContext.actor.executionReference
// resolved to empty in the pod while the legacy path filled it from the
// execution record. The same workflow behaved differently depending on which
// path started it.
func TestRunningContextFromProto_CarriesExecutionReference(t *testing.T) {
	t.Run("alongside an actor", func(t *testing.T) {
		rc := runningContextFromProto(&executionv1.ExecutionRunningContext{
			ActorType:          ptr("user"),
			ActorName:          ptr("someone"),
			ExecutionReference: ptr("exec-original"),
		})
		require.NotNil(t, rc)
		require.NotNil(t, rc.Actor)
		require.Equal(t, "someone", rc.Actor.Name)
		require.Equal(t, "exec-original", rc.Actor.ExecutionReference)
	})

	t.Run("on its own", func(t *testing.T) {
		// A rerun of an execution that had no actor still has to say what it is
		// a rerun of, so the reference alone is enough to build a context.
		rc := runningContextFromProto(&executionv1.ExecutionRunningContext{
			ExecutionReference: ptr("exec-original"),
		})
		require.NotNil(t, rc, "the reference alone must not be dropped")
		require.Equal(t, "exec-original", rc.Actor.ExecutionReference)
	})

	t.Run("still nil when there is nothing to carry", func(t *testing.T) {
		require.Nil(t, runningContextFromProto(&executionv1.ExecutionRunningContext{}))
		require.Nil(t, runningContextFromProto(nil))
	})
}

// The legacy agent path has to do the same from the execution record, and did
// not until it was noticed - see TestRerunConfigFromExecution in pkg/runner.
// Either half quietly dropping the policy runs the whole suite and says nothing.
func TestExecutionConfigFromStart_CarriesRerunPolicy(t *testing.T) {
	start := &executionv1.ExecutionStart{
		ExecutionId: ptr("exec-2"),
		Rerun: &executionv1.RerunPolicy{
			ExecutionId: ptr("exec-1"),
			OnlyFailed:  ptr(true),
			TestCases:   []string{"tests.a::test_one", "tests.b::test_two"},
		},
	}

	cfg := executionConfigFromStart(start, "org-1", nil)

	require.NotNil(t, cfg.Rerun)
	require.Equal(t, "exec-1", cfg.Rerun.ExecutionId)
	require.True(t, cfg.Rerun.OnlyFailed)
	require.Equal(t, []string{"tests.a::test_one", "tests.b::test_two"}, cfg.Rerun.TestCases)

	// The proto message may be reused by the caller, so the slice is copied.
	start.Rerun.TestCases[0] = "mutated"
	require.Equal(t, "tests.a::test_one", cfg.Rerun.TestCases[0])
}

func TestExecutionConfigFromStart_NoRerunPolicy(t *testing.T) {
	cfg := executionConfigFromStart(&executionv1.ExecutionStart{ExecutionId: ptr("exec-1")}, "org-1", nil)
	require.Nil(t, cfg.Rerun, "an ordinary execution runs everything")
}
