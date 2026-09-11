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
