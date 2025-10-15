package testresult

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"

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

	prevDate := time.Date(2023, 5, 5, 0, 0, 0, 0, time.UTC)
	midDate := prevDate.Add(time.Hour)
	nextDate := midDate.Add(time.Hour)
	testSuiteNames := []string{"test-suite-1", "test-suite-2"}
	testSuite1 := &testkube.ObjectRef{Name: testSuiteNames[0]}
	testSuite2 := &testkube.ObjectRef{Name: testSuiteNames[1]}

	startResults := []testkube.TestSuiteExecution{{Id: "id1", TestSuite: testSuite1, StartTime: midDate, EndTime: midDate}, {Id: "id2", TestSuite: testSuite2, StartTime: midDate}}
	endResults := []testkube.TestSuiteExecution{{Id: "id3", TestSuite: testSuite1, StartTime: prevDate, EndTime: nextDate}, {Id: "id4", TestSuite: testSuite2, StartTime: prevDate, EndTime: prevDate}}
	expectedResults := []testkube.TestSuiteExecution{endResults[0], startResults[1]}

	startReq := GetLatestByTestSuitesRequest{TestSuiteNames: testSuiteNames, SortField: "starttime"}
	startResponse := GetLatestByTestSuitesResponse{TestSuiteExecutions: startResults}
	startResponseBytes, _ := json.Marshal(startResponse)
	endReq := GetLatestByTestSuitesRequest{TestSuiteNames: testSuiteNames, SortField: "endtime"}
	endResponse := GetLatestByTestSuitesResponse{TestSuiteExecutions: endResults}
	endResponseBytes, _ := json.Marshal(endResponse)

	mockExecutor.EXPECT().Execute(ctx, CmdTestResultGetLatestByTestSuites, startReq).Return(startResponseBytes, nil)
	mockExecutor.EXPECT().Execute(ctx, CmdTestResultGetLatestByTestSuites, endReq).Return(endResponseBytes, nil)

	results, err := repo.GetLatestByTestSuites(ctx, testSuiteNames)
	if err != nil {
		t.Fatalf("GetLatestByTestSuites() returned an unexpected error: %v", err)
	}
	assert.Equal(t, len(results), len(expectedResults))
	assert.Contains(t, results, expectedResults[0])
	assert.Contains(t, results, expectedResults[1])
}

func TestCloudResultRepository_GetPreviousFinishedState(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := executor.NewMockExecutor(mockCtrl)
	repo := &CloudRepository{executor: mockExecutor}

	testSuiteName := "test_suite_name"
	date := time.Date(2023, 5, 5, 0, 0, 0, 0, time.UTC)
	expectedStatus := testkube.TestSuiteExecutionStatusPassed
	response, _ := json.Marshal(GetPreviousFinishedStateResponse{Result: *expectedStatus})

	mockExecutor.
		EXPECT().
		Execute(ctx, CmdTestResultGetPreviousFinishedState, GetPreviousFinishedStateRequest{TestSuiteName: testSuiteName, Date: date}).
		Return(response, nil)

	status, err := repo.GetPreviousFinishedState(ctx, testSuiteName, date)
	assert.NoError(t, err)
	assert.Equal(t, *expectedStatus, status)
}

func TestCloudResultRepository_GetNextExecutionNumber(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExecutor := executor.NewMockExecutor(ctrl)

	testSuiteName := "testsuite-1"
	var testSuiteNumber int32 = 3

	// Setup expectations for the mockedExecutor.Execute method
	expectedReq := NextExecutionNumberRequest{TestSuiteName: testSuiteName}
	expectedResponse, _ := json.Marshal(&NextExecutionNumberResponse{TestSuiteNumber: testSuiteNumber})
	mockExecutor.EXPECT().Execute(gomock.Any(), CmdTestResultGetNextExecutionNumber, expectedReq).Return(expectedResponse, nil)

	r := &CloudRepository{executor: mockExecutor}

	result, err := r.GetNextExecutionNumber(ctx, testSuiteName)
	assert.NoError(t, err)
	assert.Equal(t, testSuiteNumber, result)
}
