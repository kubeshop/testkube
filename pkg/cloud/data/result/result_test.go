package result

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

func TestCloudResultRepository_GetNextExecutionNumber(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExecutor := executor.NewMockExecutor(ctrl)

	testName := "test-1"
	var testNumber int32 = 3

	// Setup expectations for the mockedExecutor.Execute method
	expectedReq := NextExecutionNumberRequest{TestName: testName}
	expectedResponse, _ := json.Marshal(&NextExecutionNumberResponse{TestNumber: testNumber})
	mockExecutor.EXPECT().Execute(gomock.Any(), CmdResultGetNextExecutionNumber, expectedReq).Return(expectedResponse, nil)

	r := &CloudRepository{executor: mockExecutor}

	result, err := r.GetNextExecutionNumber(ctx, testName)
	assert.NoError(t, err)
	assert.Equal(t, testNumber, result)
}

func TestCloudResultRepository_Get(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	expectedExecution := testkube.Execution{Id: "id"}
	expectedResponse := GetResponse{Execution: expectedExecution}
	expectedResponseBytes, _ := json.Marshal(expectedResponse)

	mockExecutor.EXPECT().Execute(ctx, CmdResultGet, GetRequest{ID: "id"}).Return(expectedResponseBytes, nil)

	actualExecution, err := repo.Get(ctx, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Equal(t, expectedExecution.Id, actualExecution.Id)
}

func TestCloudResultRepository_GetByNameAndTest(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	expectedName := "name"
	expectedTestName := "testName"
	expectedExecution := testkube.Execution{Name: expectedName, TestName: expectedTestName}
	expectedResponse := GetByNameAndTestResponse{Execution: expectedExecution}
	expectedResponseBytes, _ := json.Marshal(expectedResponse)

	mockExecutor.
		EXPECT().
		Execute(ctx, CmdResultGetByNameAndTest, GetByNameAndTestRequest{Name: expectedName, TestName: expectedTestName}).
		Return(expectedResponseBytes, nil)

	actualExecution, err := repo.GetByNameAndTest(ctx, expectedName, expectedTestName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Equal(t, expectedExecution.Name, actualExecution.Name)
	assert.Equal(t, expectedExecution.TestName, actualExecution.TestName)
}

func TestCloudResultRepository_GetLatestByTest(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	testName := "test_name"
	sortField := "sort_field"
	execution := testkube.Execution{Id: "id"}

	req := GetLatestByTestRequest{TestName: testName, SortField: sortField}
	response := GetLatestByTestResponse{Execution: execution}
	expectedResponseBytes, _ := json.Marshal(response)
	mockExecutor.EXPECT().Execute(gomock.Any(), CmdResultGetLatestByTest, req).Return(expectedResponseBytes, nil)

	result, err := repo.GetLatestByTest(ctx, testName, sortField)
	assert.NoError(t, err)
	assert.Equal(t, execution, result)
}

func TestCloudResultRepository_Insert(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}
	result := testkube.Execution{Id: "id", Name: "name", TestName: "testName"}
	req := InsertRequest{Result: result}

	mockExecutor.EXPECT().Execute(ctx, CmdResultInsert, req).Return(nil, nil)

	err := repo.Insert(ctx, result)

	assert.NoError(t, err)
}

func TestCloudResultRepository_Update(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}
	result := testkube.Execution{Id: "id", Name: "name", TestName: "testName"}
	req := UpdateRequest{Result: result}

	mockExecutor.EXPECT().Execute(ctx, CmdResultUpdate, req).Return(nil, nil)

	err := repo.Update(ctx, result)

	assert.NoError(t, err)
}
