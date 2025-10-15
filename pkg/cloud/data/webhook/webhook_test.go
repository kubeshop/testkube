package webhook

import (
	"context"
	"encoding/json"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

func TestCloudRepository_CollectExecutionResult(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{mockExecutor}

	expectedResponse := WebhookExecutionCollectResultResponse{}
	expectedResponseBytes, _ := json.Marshal(expectedResponse)

	mockExecutor.
		EXPECT().
		Execute(context.Background(), CmdWebhookExecutionCollectResult, WebhookExecutionCollectResultRequest{WebhookName: "webhook"}).
		Return(expectedResponseBytes, nil)

	err := repo.CollectExecutionResult(context.Background(), testkube.Event{}, "webhook", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
