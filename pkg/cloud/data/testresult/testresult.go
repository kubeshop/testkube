package testresult

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
)

var _ testresult.Repository = (*CloudRepository)(nil)

type CloudRepository struct {
	executor executor.Executor
}

func NewCloudRepository(client cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudRepository {
	return &CloudRepository{executor: executor.NewCloudGRPCExecutor(client, grpcConn, apiKey)}
}

func (r *CloudRepository) Get(ctx context.Context, id string) (testkube.TestSuiteExecution, error) {
	req := GetRequest{ID: id}
	response, err := r.executor.Execute(ctx, CmdTestResultGet, req)
	if err != nil {
		return testkube.TestSuiteExecution{}, err
	}
	var commandResponse GetResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.TestSuiteExecution{}, err
	}
	return commandResponse.TestSuiteExecution, nil
}

func (r *CloudRepository) GetByNameAndTestSuite(ctx context.Context, name, testSuiteName string) (testkube.TestSuiteExecution, error) {
	req := GetByNameAndTestSuiteRequest{Name: name, TestSuiteName: testSuiteName}
	response, err := r.executor.Execute(ctx, CmdTestResultGetByNameAndTestSuite, req)
	if err != nil {
		return testkube.TestSuiteExecution{}, err
	}
	var commandResponse GetByNameAndTestSuiteResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.TestSuiteExecution{}, err
	}
	return commandResponse.TestSuiteExecution, nil
}

func (r *CloudRepository) getLatestByTestSuite(ctx context.Context, testSuiteName, sortField string) (testkube.TestSuiteExecution, error) {
	req := GetLatestByTestSuiteRequest{TestSuiteName: testSuiteName, SortField: sortField}
	response, err := r.executor.Execute(ctx, CmdTestResultGetLatestByTestSuite, req)
	if err != nil {
		return testkube.TestSuiteExecution{}, err
	}
	var commandResponse GetLatestByTestSuiteResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.TestSuiteExecution{}, err
	}
	return commandResponse.TestSuiteExecution, nil
}

// TODO: When it will be implemented, replace with a new Cloud command, to avoid 2 calls with 2 sort fields
func (r *CloudRepository) GetLatestByTestSuite(ctx context.Context, testSuiteName string) (*testkube.TestSuiteExecution, error) {
	startExecution, startErr := r.getLatestByTestSuite(ctx, testSuiteName, "starttime")
	if startErr != nil && startErr != mongo.ErrNoDocuments {
		return nil, startErr
	}
	endExecution, endErr := r.getLatestByTestSuite(ctx, testSuiteName, "endtime")
	if endErr != nil && endErr != mongo.ErrNoDocuments {
		return nil, endErr
	}

	if startErr == nil && endErr == nil {
		if startExecution.StartTime.After(endExecution.EndTime) {
			return &startExecution, nil
		} else {
			return &endExecution, nil
		}
	} else if startErr == nil {
		return &startExecution, nil
	} else if endErr == nil {
		return &endExecution, nil
	}
	return nil, startErr
}

func (r *CloudRepository) getLatestByTestSuites(ctx context.Context, testSuiteNames []string, sortField string) (executions []testkube.TestSuiteExecution, err error) {
	req := GetLatestByTestSuitesRequest{TestSuiteNames: testSuiteNames, SortField: sortField}
	response, err := r.executor.Execute(ctx, CmdTestResultGetLatestByTestSuites, req)
	if err != nil {
		return nil, err
	}
	var commandResponse GetLatestByTestSuitesResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	return commandResponse.TestSuiteExecutions, nil
}

// TODO: When it will be implemented, replace with a new Cloud command, to avoid 2 calls with 2 sort fields
func (r *CloudRepository) GetLatestByTestSuites(ctx context.Context, testSuiteNames []string) ([]testkube.TestSuiteExecution, error) {
	startExecutions, err := r.getLatestByTestSuites(ctx, testSuiteNames, "starttime")
	if err != nil {
		return nil, err
	}
	endExecutions, err := r.getLatestByTestSuites(ctx, testSuiteNames, "endtime")
	if err != nil {
		return nil, err
	}
	executionsCount := len(startExecutions)
	if len(endExecutions) > executionsCount {
		executionsCount = len(endExecutions)
	}
	executionsMap := make(map[string]*testkube.TestSuiteExecution, executionsCount)
	for i := range startExecutions {
		if startExecutions[i].TestSuite == nil {
			continue
		}
		executionsMap[startExecutions[i].TestSuite.Name] = &startExecutions[i]
	}
	for i := range endExecutions {
		if endExecutions[i].TestSuite == nil {
			continue
		}
		startExecution, ok := executionsMap[endExecutions[i].TestSuite.Name]
		if ok {
			if endExecutions[i].EndTime.After(startExecution.StartTime) {
				executionsMap[endExecutions[i].TestSuite.Name] = &endExecutions[i]
			}
		} else {
			executionsMap[endExecutions[i].TestSuite.Name] = &endExecutions[i]
		}
	}
	executions := make([]testkube.TestSuiteExecution, 0, executionsCount)
	for _, value := range executionsMap {
		executions = append(executions, *value)
	}
	return executions, nil
}

func (r *CloudRepository) GetExecutionsTotals(ctx context.Context, filters ...testresult.Filter) (totals testkube.ExecutionsTotals, err error) {
	var filterImpls []*testresult.FilterImpl
	for _, f := range filters {
		filterImpl, ok := f.(*testresult.FilterImpl)
		if !ok {
			return testkube.ExecutionsTotals{}, errors.New("invalid filter")
		}
		filterImpls = append(filterImpls, filterImpl)
	}
	req := GetExecutionsTotalsRequest{Filter: filterImpls}
	response, err := r.executor.Execute(ctx, CmdTestResultGetExecutionsTotals, req)
	if err != nil {
		return testkube.ExecutionsTotals{}, err
	}
	var commandResponse GetExecutionsTotalsResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.ExecutionsTotals{}, err
	}
	return commandResponse.ExecutionsTotals, nil
}

func (r *CloudRepository) GetExecutions(ctx context.Context, filter testresult.Filter) ([]testkube.TestSuiteExecution, error) {
	filterImpl, ok := filter.(*testresult.FilterImpl)
	if !ok {
		return nil, errors.New("invalid filter")
	}
	req := GetExecutionsRequest{Filter: filterImpl}
	response, err := r.executor.Execute(ctx, CmdTestResultGetExecutions, req)
	if err != nil {
		return nil, err
	}
	var commandResponse GetExecutionsResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	return commandResponse.TestSuiteExecutions, nil
}

func (r *CloudRepository) Insert(ctx context.Context, result testkube.TestSuiteExecution) error {
	req := InsertRequest{TestSuiteExecution: result}
	_, err := r.executor.Execute(ctx, CmdTestResultInsert, req)
	return err
}

func (r *CloudRepository) Update(ctx context.Context, result testkube.TestSuiteExecution) error {
	req := UpdateRequest{TestSuiteExecution: result}
	_, err := r.executor.Execute(ctx, CmdTestResultUpdate, req)
	return err
}

func (r *CloudRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	req := StartExecutionRequest{ID: id, StartTime: startTime}
	_, err := r.executor.Execute(ctx, CmdTestResultStartExecution, req)
	return err
}

func (r *CloudRepository) EndExecution(ctx context.Context, execution testkube.TestSuiteExecution) error {
	req := EndExecutionRequest{Execution: execution}
	_, err := r.executor.Execute(ctx, CmdTestResultEndExecution, req)
	return err
}

func (r *CloudRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	req := DeleteByTestSuiteRequest{TestSuiteName: testSuiteName}
	_, err := r.executor.Execute(ctx, CmdTestResultDeleteByTestSuite, req)
	return err
}

func (r *CloudRepository) DeleteAll(ctx context.Context) error {
	_, err := r.executor.Execute(ctx, CmdTestResultDeleteAll, nil)
	return err
}

func (r *CloudRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) error {
	req := DeleteByTestSuitesRequest{TestSuiteNames: testSuiteNames}
	_, err := r.executor.Execute(ctx, CmdTestResultDeleteByTestSuites, req)
	return err
}

func (r *CloudRepository) GetTestSuiteMetrics(ctx context.Context, name string, limit, last int) (testkube.ExecutionsMetrics, error) {
	req := GetTestSuiteMetricsRequest{Name: name, Limit: limit, Last: last}
	response, err := r.executor.Execute(ctx, CmdTestResultGetTestSuiteMetrics, req)
	if err != nil {
		return testkube.ExecutionsMetrics{}, err
	}
	var commandResponse GetTestSuiteMetricsResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.ExecutionsMetrics{}, err
	}
	return commandResponse.Metrics, nil
}

func (r *CloudRepository) Count(ctx context.Context, filter testresult.Filter) (int64, error) {
	return 0, nil
}
