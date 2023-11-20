package config

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

var ctx = context.Background()

func TestCloudRepository_GetUniqueClusterId(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExecutor := executor.NewMockExecutor(ctrl)

	// Setup expectations for the mockedExecutor.Execute method
	expectedReq := GetUniqueClusterIdRequest{}
	expectedResponse, _ := json.Marshal(&GetUniqueClusterIdResponse{ClusterID: "test-cluster"})
	mockExecutor.EXPECT().Execute(gomock.Any(), CmdConfigGetUniqueClusterId, expectedReq).Return(expectedResponse, nil)

	r := &CloudRepository{executor: mockExecutor}

	result, err := r.GetUniqueClusterId(ctx)
	assert.NoError(t, err)
	assert.Equal(t, result, "test-cluster")
}

func TestCloudRepository_GetTelemetryEnabled(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExecutor := executor.NewMockExecutor(ctrl)

	// Setup expectations for the mockedExecutor.Execute method
	expectedReq := GetTelemetryEnabledRequest{}
	expectedResponse, _ := json.Marshal(&GetTelemetryEnabledResponse{Enabled: true})
	mockExecutor.EXPECT().Execute(gomock.Any(), CmdConfigGetTelemetryEnabled, expectedReq).Return(expectedResponse, nil)

	r := &CloudRepository{executor: mockExecutor}

	result, err := r.GetTelemetryEnabled(ctx)
	assert.NoError(t, err)
	assert.Equal(t, result, true)
}

func TestCloudRepository_Get(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	expectedConfig := testkube.Config{Id: "test-id", ClusterId: "test-cluster", EnableTelemetry: true}
	expectedResponse := GetResponse{Config: expectedConfig}
	expectedResponseBytes, _ := json.Marshal(expectedResponse)

	mockExecutor.EXPECT().Execute(ctx, CmdConfigGet, GetRequest{}).Return(expectedResponseBytes, nil)

	actualConfig, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Equal(t, expectedConfig, actualConfig)
}

func TestCloudRepository_Upsert(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	expectedConfig := testkube.Config{Id: "test-id2", ClusterId: "test-cluster2", EnableTelemetry: true}
	expectedResponse := UpsertResponse{Config: expectedConfig}
	expectedResponseBytes, _ := json.Marshal(expectedResponse)

	mockExecutor.EXPECT().Execute(ctx, CmdConfigUpsert, UpsertRequest{Config: expectedConfig}).Return(expectedResponseBytes, nil)

	actualConfig, err := repo.Upsert(ctx, expectedConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Equal(t, expectedConfig, actualConfig)
}
