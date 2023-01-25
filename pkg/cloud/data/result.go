package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/repository/result"

	"github.com/kubeshop/testkube/pkg/cloud"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var _ result.Repository = (*CloudResultRepository)(nil)

type CloudResultRepository struct {
	cloudClient cloud.TestKubeCloudAPIClient
	apiKey      string
}

func NewCloudResultRepository(cloudClient cloud.TestKubeCloudAPIClient, apiKey string) *CloudResultRepository {
	return &CloudResultRepository{cloudClient: cloudClient, apiKey: apiKey}
}

func (r *CloudResultRepository) GetNextExecutionNumber(ctx context.Context, testName string) (int32, error) {
	req := NextExecutionNumberResultRequest{TestName: testName}
	response, err := execute(ctx, r.cloudClient, CmdResultGetNextExecutionNumber, req, r.apiKey)
	if err != nil {
		return 0, err
	}
	var commandResponse NextExecutionNumberResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return 0, err
	}
	return commandResponse.TestNumber, nil
}

func (r *CloudResultRepository) Get(ctx context.Context, id string) (testkube.Execution, error) {
	req := GetResultRequest{ID: id}
	response, err := execute(ctx, r.cloudClient, CmdResultGet, req, r.apiKey)
	if err != nil {
		return testkube.Execution{}, err
	}
	var commandResponse GetResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return testkube.Execution{}, err
	}
	return commandResponse.Execution, nil
}

func (r *CloudResultRepository) GetByNameAndTest(ctx context.Context, name, testName string) (testkube.Execution, error) {
	req := GetByNameAndTestResultRequest{Name: name, TestName: testName}
	response, err := execute(ctx, r.cloudClient, CmdResultGetByNameAndTest, req, r.apiKey)
	if err != nil {
		return testkube.Execution{}, err
	}
	var commandResponse GetByNameAndTestResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return testkube.Execution{}, err
	}
	return commandResponse.Execution, nil
}

func (r *CloudResultRepository) GetLatestByTest(ctx context.Context, testName, sortField string) (testkube.Execution, error) {
	req := GetLatestByTestResultRequest{TestName: testName, SortField: sortField}
	response, err := execute(ctx, r.cloudClient, CmdResultGetLatestByTest, req, r.apiKey)
	if err != nil {
		return testkube.Execution{}, err
	}
	var commandResponse GetLatestByTestResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return testkube.Execution{}, err
	}
	return commandResponse.Execution, nil
}

func (r *CloudResultRepository) GetLatestByTests(ctx context.Context, testNames []string, sortField string) ([]testkube.Execution, error) {
	req := GetLatestByTestsResultRequest{TestNames: testNames, SortField: sortField}
	response, err := execute(ctx, r.cloudClient, CmdResultGetLatestByTests, req, r.apiKey)
	if err != nil {
		return nil, err
	}
	var commandResponse GetLatestByTestsResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return nil, err
	}
	return commandResponse.Executions, nil
}

func (r *CloudResultRepository) GetExecutions(ctx context.Context, filter result.Filter) ([]testkube.Execution, error) {
	filterImpl, ok := filter.(*result.FilterImpl)
	if !ok {
		return nil, errors.New("invalid filter")
	}
	req := GetExecutionsResultRequest{Filter: filterImpl}
	response, err := execute(ctx, r.cloudClient, CmdResultGetExecutions, req, r.apiKey)
	if err != nil {
		return nil, err
	}
	var commandResponse GetLatestByTestsResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return nil, err
	}
	return commandResponse.Executions, nil
}

func (r *CloudResultRepository) GetExecutionTotals(ctx context.Context, paging bool, filters ...result.Filter) (testkube.ExecutionsTotals, error) {
	var filterImpls []*result.FilterImpl
	for _, f := range filters {
		filterImpl, ok := f.(*result.FilterImpl)
		if !ok {
			return testkube.ExecutionsTotals{}, errors.New("invalid filter")
		}
		filterImpls = append(filterImpls, filterImpl)
	}
	req := GetExecutionTotalsResultRequest{Paging: paging, Filter: filterImpls}
	response, err := execute(ctx, r.cloudClient, CmdResultGetExecutionTotals, req, r.apiKey)
	if err != nil {
		return testkube.ExecutionsTotals{}, err
	}
	var commandResponse GetExecutionTotalsResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return testkube.ExecutionsTotals{}, err
	}
	return commandResponse.Result, nil
}

func (r *CloudResultRepository) Insert(ctx context.Context, result testkube.Execution) error {
	req := InsertResultRequest{Result: result}
	_, err := execute(ctx, r.cloudClient, CmdResultInsert, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) Update(ctx context.Context, result testkube.Execution) error {
	req := UpdateResultRequest{Result: result}
	_, err := execute(ctx, r.cloudClient, CmdResultUpdate, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) UpdateResult(ctx context.Context, id string, execution testkube.Execution) error {
	req := UpdateResultInExecutionResultRequest{ID: id, Execution: execution}
	_, err := execute(ctx, r.cloudClient, CmdResultUpdateResult, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	req := StartExecutionResultRequest{ID: id, StartTime: startTime}
	_, err := execute(ctx, r.cloudClient, CmdResultStartExecution, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) EndExecution(ctx context.Context, execution testkube.Execution) error {
	req := EndExecutionResultRequest{Execution: execution}
	_, err := execute(ctx, r.cloudClient, CmdResultEndExecution, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) GetLabels(ctx context.Context) (map[string][]string, error) {
	response, err := execute(ctx, r.cloudClient, CmdResultGetLabels, nil, r.apiKey)
	if err != nil {
		return nil, err
	}
	var commandResponse GetLabelsResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *CloudResultRepository) DeleteByTest(ctx context.Context, testName string) error {
	req := DeleteByTestResultRequest{TestName: testName}
	_, err := execute(ctx, r.cloudClient, CmdResultDeleteByTest, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	req := DeleteByTestSuiteResultRequest{TestSuiteName: testSuiteName}
	_, err := execute(ctx, r.cloudClient, CmdResultDeleteByTestSuite, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) DeleteAll(ctx context.Context) error {
	req := DeleteAllResultRequest{}
	_, err := execute(ctx, r.cloudClient, CmdResultDeleteAll, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) DeleteByTests(ctx context.Context, testNames []string) error {
	req := DeleteByTestsResultRequest{TestNames: testNames}
	_, err := execute(ctx, r.cloudClient, CmdResultDeleteByTests, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) error {
	req := DeleteByTestSuitesResultRequest{TestSuiteNames: testSuiteNames}
	_, err := execute(ctx, r.cloudClient, CmdResultDeleteByTestSuites, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) DeleteForAllTestSuites(ctx context.Context) error {
	req := DeleteForAllTestSuitesResultResponse{}
	_, err := execute(ctx, r.cloudClient, CmdResultDeleteForAllTestSuites, req, r.apiKey)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudResultRepository) GetTestMetrics(ctx context.Context, name string, limit, last int) (testkube.ExecutionsMetrics, error) {
	req := GetTestMetricsResultRequest{Name: name, Limit: limit, Last: last}
	response, err := execute(ctx, r.cloudClient, CmdResultGetTestMetrics, req, r.apiKey)
	if err != nil {
		return testkube.ExecutionsMetrics{}, err
	}
	var commandResponse GetTestMetricsResultResponse
	if err := json.Unmarshal(response.Response, &commandResponse); err != nil {
		return testkube.ExecutionsMetrics{}, err
	}
	return commandResponse.Metrics, nil
}
