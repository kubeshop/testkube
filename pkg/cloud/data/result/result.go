package result

import (
	"context"
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/cloud/data/executor"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/repository/result"

	"github.com/kubeshop/testkube/pkg/cloud"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var _ result.Repository = (*CloudRepository)(nil)

type CloudRepository struct {
	executor executor.Executor
}

func NewCloudResultRepository(cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudRepository {
	return &CloudRepository{executor: executor.NewCloudGRPCExecutor(cloudClient, grpcConn, apiKey)}
}

func (r *CloudRepository) GetNextExecutionNumber(ctx context.Context, testName string) (int32, error) {
	req := NextExecutionNumberRequest{TestName: testName}
	response, err := r.executor.Execute(ctx, CmdResultGetNextExecutionNumber, req)
	if err != nil {
		return 0, err
	}
	var commandResponse NextExecutionNumberResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return 0, err
	}
	return commandResponse.TestNumber, nil
}

func (r *CloudRepository) GetExecution(ctx context.Context, id string) (testkube.Execution, error) {
	req := GetRequest{ID: id}
	response, err := r.executor.Execute(ctx, CmdResultGet, req)
	if err != nil {
		return testkube.Execution{}, err
	}
	var commandResponse GetResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.Execution{}, err
	}
	return commandResponse.Execution, nil
}

func (r *CloudRepository) Get(ctx context.Context, id string) (testkube.Execution, error) {
	req := GetRequest{ID: id}
	response, err := r.executor.Execute(ctx, CmdResultGet, req)
	if err != nil {
		return testkube.Execution{}, err
	}
	var commandResponse GetResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.Execution{}, err
	}
	return commandResponse.Execution, nil
}

func (r *CloudRepository) GetByNameAndTest(ctx context.Context, name, testName string) (testkube.Execution, error) {
	req := GetByNameAndTestRequest{Name: name, TestName: testName}
	response, err := r.executor.Execute(ctx, CmdResultGetByNameAndTest, req)
	if err != nil {
		return testkube.Execution{}, err
	}
	var commandResponse GetByNameAndTestResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.Execution{}, err
	}
	return commandResponse.Execution, nil
}

func (r *CloudRepository) getLatestByTest(ctx context.Context, testName, sortField string) (testkube.Execution, error) {
	req := GetLatestByTestRequest{TestName: testName, SortField: sortField}
	response, err := r.executor.Execute(ctx, CmdResultGetLatestByTest, req)
	if err != nil {
		return testkube.Execution{}, err
	}
	var commandResponse GetLatestByTestResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.Execution{}, err
	}
	return commandResponse.Execution, nil
}

// TODO: When it will be implemented, replace with a new Cloud command, to avoid 2 calls with 2 sort fields
func (r *CloudRepository) GetLatestByTest(ctx context.Context, testName string) (*testkube.Execution, error) {
	startExecution, startErr := r.getLatestByTest(ctx, testName, "starttime")
	if startErr != nil && startErr != mongo.ErrNoDocuments {
		return nil, startErr
	}
	endExecution, endErr := r.getLatestByTest(ctx, testName, "endtime")
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

// TODO: When it will be implemented, replace with a new Cloud command, to avoid 2 calls with 2 sort fields
func (r *CloudRepository) getLatestByTests(ctx context.Context, testNames []string, sortField string) ([]testkube.Execution, error) {
	req := GetLatestByTestsRequest{TestNames: testNames, SortField: sortField}
	response, err := r.executor.Execute(ctx, CmdResultGetLatestByTests, req)
	if err != nil {
		return nil, err
	}
	var commandResponse GetLatestByTestsResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	return commandResponse.Executions, nil
}

// TODO: When it will be implemented, replace with a new Cloud command, to avoid 2 calls with 2 sort fields
func (r *CloudRepository) GetLatestByTests(ctx context.Context, testNames []string) ([]testkube.Execution, error) {
	startExecutions, err := r.getLatestByTests(ctx, testNames, "starttime")
	if err != nil {
		return nil, err
	}
	endExecutions, err := r.getLatestByTests(ctx, testNames, "endtime")
	if err != nil {
		return nil, err
	}
	executionsCount := len(startExecutions)
	if len(endExecutions) > executionsCount {
		executionsCount = len(endExecutions)
	}
	executionsMap := make(map[string]*testkube.Execution, executionsCount)
	for i := range startExecutions {
		executionsMap[startExecutions[i].TestName] = &startExecutions[i]
	}
	for i := range endExecutions {
		startExecution, ok := executionsMap[endExecutions[i].TestName]
		if ok {
			if endExecutions[i].EndTime.After(startExecution.StartTime) {
				executionsMap[endExecutions[i].TestName] = &endExecutions[i]
			}
		} else {
			executionsMap[endExecutions[i].TestName] = &endExecutions[i]
		}
	}
	executions := make([]testkube.Execution, 0, executionsCount)
	for _, value := range executionsMap {
		executions = append(executions, *value)
	}
	return executions, nil
}

func (r *CloudRepository) GetExecutions(ctx context.Context, filter result.Filter) ([]testkube.Execution, error) {
	filterImpl, ok := filter.(*result.FilterImpl)
	if !ok {
		return nil, errors.New("invalid filter")
	}
	req := GetExecutionsRequest{Filter: filterImpl}
	response, err := r.executor.Execute(ctx, CmdResultGetExecutions, req)
	if err != nil {
		return nil, err
	}
	var commandResponse GetLatestByTestsResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	return commandResponse.Executions, nil
}

func (r *CloudRepository) GetExecutionTotals(ctx context.Context, paging bool, filters ...result.Filter) (testkube.ExecutionsTotals, error) {
	var filterImpls []*result.FilterImpl
	for _, f := range filters {
		filterImpl, ok := f.(*result.FilterImpl)
		if !ok {
			return testkube.ExecutionsTotals{}, errors.New("invalid filter")
		}
		filterImpls = append(filterImpls, filterImpl)
	}
	req := GetExecutionTotalsRequest{Paging: paging, Filter: filterImpls}
	response, err := r.executor.Execute(ctx, CmdResultGetExecutionTotals, req)
	if err != nil {
		return testkube.ExecutionsTotals{}, err
	}
	var commandResponse GetExecutionTotalsResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.ExecutionsTotals{}, err
	}
	return commandResponse.Result, nil
}

func (r *CloudRepository) Insert(ctx context.Context, result testkube.Execution) error {
	req := InsertRequest{Result: result}
	_, err := r.executor.Execute(ctx, CmdResultInsert, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) Update(ctx context.Context, result testkube.Execution) error {
	req := UpdateRequest{Result: result}
	_, err := r.executor.Execute(ctx, CmdResultUpdate, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) UpdateResult(ctx context.Context, id string, execution testkube.Execution) error {
	req := UpdateResultInExecutionRequest{ID: id, Execution: execution}
	_, err := r.executor.Execute(ctx, CmdResultUpdateResult, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	req := StartExecutionRequest{ID: id, StartTime: startTime}
	_, err := r.executor.Execute(ctx, CmdResultStartExecution, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) EndExecution(ctx context.Context, execution testkube.Execution) error {
	req := EndExecutionRequest{Execution: execution}
	_, err := r.executor.Execute(ctx, CmdResultEndExecution, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) GetLabels(ctx context.Context) (map[string][]string, error) {
	response, err := r.executor.Execute(ctx, CmdResultGetLabels, nil)
	if err != nil {
		return nil, err
	}
	var commandResponse GetLabelsResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *CloudRepository) DeleteByTest(ctx context.Context, testName string) error {
	req := DeleteByTestRequest{TestName: testName}
	_, err := r.executor.Execute(ctx, CmdResultDeleteByTest, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	req := DeleteByTestSuiteRequest{TestSuiteName: testSuiteName}
	_, err := r.executor.Execute(ctx, CmdResultDeleteByTestSuite, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) DeleteAll(ctx context.Context) error {
	req := DeleteAllRequest{}
	_, err := r.executor.Execute(ctx, CmdResultDeleteAll, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) DeleteByTests(ctx context.Context, testNames []string) error {
	req := DeleteByTestsRequest{TestNames: testNames}
	_, err := r.executor.Execute(ctx, CmdResultDeleteByTests, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) error {
	req := DeleteByTestSuitesRequest{TestSuiteNames: testSuiteNames}
	_, err := r.executor.Execute(ctx, CmdResultDeleteByTestSuites, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) DeleteForAllTestSuites(ctx context.Context) error {
	req := DeleteForAllTestSuitesResponse{}
	_, err := r.executor.Execute(ctx, CmdResultDeleteForAllTestSuites, req)
	if err != nil {
		return err
	}
	return nil
}

func (r *CloudRepository) GetTestMetrics(ctx context.Context, name string, limit, last int) (testkube.ExecutionsMetrics, error) {
	req := GetTestMetricsRequest{Name: name, Limit: limit, Last: last}
	response, err := r.executor.Execute(ctx, CmdResultGetTestMetrics, req)
	if err != nil {
		return testkube.ExecutionsMetrics{}, err
	}
	var commandResponse GetTestMetricsResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return testkube.ExecutionsMetrics{}, err
	}
	return commandResponse.Metrics, nil
}

func (r *CloudRepository) Count(ctx context.Context, filter result.Filter) (int64, error) {
	return 0, nil
}
