package grpc_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
)

func syncedBoard() *insightsv1.Board {
	board := declarative.Kinds()[0].New().(*insightsv1.Board)
	board.ObjectMeta = metav1.ObjectMeta{
		Name:            "release-health",
		Namespace:       "testkube",
		ResourceVersion: "42",
		Annotations:     map[string]string{"testkube.io/gitops-owner": "agent-1"},
	}
	board.Spec.DisplayName = "Release health"
	board.Status = commonv1.ControlPlaneResourceStatus{ID: "board-1", GitOpsOwner: &commonv1.GitOpsOwner{AgentID: "forged"}}
	return board
}

// A Control Plane resource is sent as a PUT to the declarative endpoints would send it: named by
// its kind's resource and its name, with what the cluster manages and its status left out, and the
// annotations that carry ownership kept.
func TestUpdateOrCreateControlPlaneResource(t *testing.T) {
	var srv testSrv
	client := startGRPCTestConnection(t, &srv)

	require.NoError(t, client.UpdateOrCreateControlPlaneResource(t.Context(), syncedBoard()))

	got := srv.ControlPlaneResource
	require.NotNil(t, got)
	assert.Equal(t, "boards", got.GetResource())
	assert.Equal(t, "release-health", got.GetName())
	var sent map[string]any
	require.NoError(t, json.Unmarshal(got.GetPayload(), &sent))
	assert.Equal(t, "insights.testkube.io/v1alpha1", sent["apiVersion"])
	assert.Equal(t, "Board", sent["kind"])
	assert.Equal(t, map[string]any{
		"name":        "release-health",
		"annotations": map[string]any{"testkube.io/gitops-owner": "agent-1"},
	}, sent["metadata"])
	assert.NotContains(t, sent, "status", "ownership is never taken from the payload")
}

func TestDeleteControlPlaneResource(t *testing.T) {
	var srv testSrv
	client := startGRPCTestConnection(t, &srv)

	require.NoError(t, client.DeleteControlPlaneResource(t.Context(), declarative.Kinds()[0], "release-health"))

	got := srv.Deleted.GetControlPlaneResource()
	require.NotNil(t, got)
	assert.Equal(t, "boards", got.GetResource())
	assert.Equal(t, "release-health", got.GetName())
}

// A rejection is translated as for every other kind, so the reconciler stops retrying it.
func TestControlPlaneResourceRejection(t *testing.T) {
	srv := testSrv{Err: status.Error(codes.FailedPrecondition, `owned by "prod-sync" (agent-2)`)}
	client := startGRPCTestConnection(t, &srv)

	err := client.UpdateOrCreateControlPlaneResource(t.Context(), syncedBoard())
	assert.True(t, errors.Is(err, syncagent.ErrOwnershipConflict), err)
	err = client.DeleteControlPlaneResource(t.Context(), declarative.Kinds()[0], "release-health")
	assert.True(t, errors.Is(err, syncagent.ErrOwnershipConflict), err)

	srv.Err = status.Error(codes.InvalidArgument, "spec.reports[0].kind: Unsupported value")
	err = client.UpdateOrCreateControlPlaneResource(t.Context(), syncedBoard())
	assert.True(t, errors.Is(err, syncagent.ErrInvalidResource), err)
}
