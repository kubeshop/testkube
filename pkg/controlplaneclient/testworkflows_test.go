package controlplaneclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
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

func TestNotificationStreamSessionManagerReplaysAfterCursor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	manager := newNotificationStreamSessionManager(
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "one"})
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "two"})
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "three"})
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	session, sub, replay, available, lastSeqNo, done := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1"})
	require.True(t, available)
	require.False(t, done)
	require.Empty(t, replay)

	var firstPass []uint32
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event, ok := <-sub.ch:
			if !ok {
				session.unsubscribe(sub)
				goto reconnect
			}
			firstPass = append(firstPass, event.seqNo)
		case <-deadline:
			t.Fatal("timed out waiting for initial stream to finish")
		}
	}

reconnect:
	assert.Equal(t, []uint32{1, 2, 3}, firstPass)

	session, sub, replay, available, lastSeqNo, done = manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", ResumeAfterSeqNo: 1})
	require.True(t, available)
	require.True(t, done)
	require.Equal(t, uint32(3), lastSeqNo)
	require.Len(t, replay, 2)
	assert.Equal(t, []uint32{2, 3}, []uint32{replay[0].seqNo, replay[1].seqNo})
	session.unsubscribe(sub)
}
