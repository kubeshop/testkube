package testresult

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

func TestCloudResultRepository_Get(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	expectedExecution := testkube.TestSuiteExecution{Id: "id1"}
	expectedResponse := GetResponse{TestSuiteExecution: expectedExecution}
	expectedResponseBytes, _ := json.Marshal(expectedResponse)
	mockExecutor.EXPECT().Execute(ctx, CmdTestResultGet, GetRequest{"id1"}).Return(expectedResponseBytes, nil)

	execution, err := repo.Get(ctx, "id1")
	if err != nil {
		t.Fatalf("Get() returned an unexpected error: %v", err)
	}

	assert.Equal(t, expectedExecution.Id, execution.Id)
}

func TestCloudResultRepository_GetByNameAndTestSuite(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	expectedExecution := testkube.TestSuiteExecution{Name: "name1"}
	expectedResponse := GetByNameAndTestSuiteResponse{TestSuiteExecution: expectedExecution}
	expectedResponseBytes, _ := json.Marshal(expectedResponse)
	mockExecutor.
		EXPECT().
		Execute(ctx, CmdTestResultGetByNameAndTestSuite, GetByNameAndTestSuiteRequest{"name1", "testsuite1"}).
		Return(expectedResponseBytes, nil)

	execution, err := repo.GetByNameAndTestSuite(ctx, "name1", "testsuite1")
	if err != nil {
		t.Fatalf("GetByNameAndTestSuite() returned an unexpected error: %v", err)
	}

	assert.Equal(t, expectedExecution.Name, execution.Name)
}

func TestCloudResultRepository_GetLatestByTestSuites(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	testSuiteNames := []string{"test-suite-1", "test-suite-2"}
	sortField := "sort-field"
	expectedResults := []testkube.TestSuiteExecution{{Id: "id1"}, {Id: "id2"}}
	req := GetLatestByTestSuitesRequest{TestSuiteNames: testSuiteNames, SortField: sortField}
	expectedResponse := GetLatestByTestSuitesResponse{TestSuiteExecutions: expectedResults}
	expectedResponseBytes, _ := json.Marshal(expectedResponse)

	mockExecutor.EXPECT().Execute(ctx, CmdTestResultGetLatestByTestSuites, req).Return(expectedResponseBytes, nil)

	results, err := repo.GetLatestByTestSuites(ctx, testSuiteNames, sortField)
	if err != nil {
		t.Fatalf("GetLatestByTestSuites() returned an unexpected error: %v", err)
	}
	assert.Equal(t, expectedResults, results)
}
