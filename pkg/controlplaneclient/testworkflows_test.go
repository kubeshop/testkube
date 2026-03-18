package controlplaneclient

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestListTestWorkflows_ForwardsOptionsLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockCloudClient := cloud.NewMockTestKubeCloudAPIClient(ctrl)
	client := &client{
		client:     mockCloudClient,
		proContext: config.ProContext{},
	}

	expectedErr := errors.New("list failed")
	mockCloudClient.EXPECT().
		ListTestWorkflows(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *cloud.ListTestWorkflowsRequest, _ ...grpc.CallOption) (cloud.TestKubeCloudAPI_ListTestWorkflowsClient, error) {
			require.Equal(t, uint32(25), req.Offset)
			require.Equal(t, uint32(250), req.Limit)
			require.Equal(t, map[string]string{"team": "qa"}, req.Labels)
			require.Equal(t, "smoke", req.TextSearch)
			return nil, expectedErr
		})

	items, err := client.ListTestWorkflows(context.Background(), "env-1", ListTestWorkflowOptions{
		Offset:     25,
		Limit:      250,
		Labels:     map[string]string{"team": "qa"},
		TextSearch: "smoke",
	}).All()

	require.ErrorIs(t, err, expectedErr)
	require.Empty(t, items)
}

func TestListTestWorkflows_LeavesLimitUnsetWhenOptionIsZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockCloudClient := cloud.NewMockTestKubeCloudAPIClient(ctrl)
	client := &client{
		client:     mockCloudClient,
		proContext: config.ProContext{},
	}

	expectedErr := errors.New("list failed")
	mockCloudClient.EXPECT().
		ListTestWorkflows(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *cloud.ListTestWorkflowsRequest, _ ...grpc.CallOption) (cloud.TestKubeCloudAPI_ListTestWorkflowsClient, error) {
			require.Equal(t, uint32(0), req.Limit)
			return nil, expectedErr
		})

	_, err := client.ListTestWorkflows(context.Background(), "env-1", ListTestWorkflowOptions{}).All()

	require.ErrorIs(t, err, expectedErr)
}
