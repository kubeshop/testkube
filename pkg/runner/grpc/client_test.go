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

	cfg := executionConfigFromStart(start, "org-1")

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
