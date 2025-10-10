package controlplane

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudconfig "github.com/kubeshop/testkube/pkg/cloud/data/config"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	cloudresult "github.com/kubeshop/testkube/pkg/cloud/data/result"
	cloudtestresult "github.com/kubeshop/testkube/pkg/cloud/data/testresult"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	miniorepo "github.com/kubeshop/testkube/pkg/repository/testworkflow/minio"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
)

func (s *Server) Call(ctx context.Context, request *cloud.CommandRequest) (*cloud.CommandResponse, error) {
	if cmd, ok := s.commands[cloudexecutor.Command(request.Command)]; ok {
		return cmd(ctx, request)
	}
	return nil, CommandNotImplementedError(request.Command)
}

func CreateCommands(disableDeprecatedTests bool, storageBucket string, deprecatedRepositories commons.DeprecatedRepositories, storageClient domainstorage.Client, testWorkflowOutputRepository *miniorepo.MinioRepository, testWorkflowResultsRepository testworkflow.Repository, artifactStorage *minio.ArtifactClient) []CommandHandlers {
	// Set up "Config" commands
	configCommands := CommandHandlers{
		cloudconfig.CmdConfigGetOrganizationPlan: Handler(func(ctx context.Context, data checktcl.GetOrganizationPlanRequest) (r checktcl.GetOrganizationPlanResponse, err error) {
			return
		}),
	}

	// Set up "Webhook commands
	webhoookCommands := CommandHandlers{
		cloudwebhook.CmdWebhookExecutionCollectResult: Handler(func(ctx context.Context, data cloudwebhook.WebhookExecutionCollectResultRequest) (r cloudwebhook.WebhookExecutionCollectResultResponse, err error) {
			return
		}),
	}

	// Set up "Tests - Executions" commands
	deprecatedTestExecutionsCommands := CommandHandlers{
		cloudresult.CmdResultGet: Handler(func(ctx context.Context, data cloudresult.GetRequest) (r cloudresult.GetResponse, err error) {
			r.Execution, err = deprecatedRepositories.TestResults().Get(ctx, data.ID)
			return
		}),
		cloudresult.CmdResultGetByNameAndTest: Handler(func(ctx context.Context, data cloudresult.GetByNameAndTestRequest) (r cloudresult.GetByNameAndTestResponse, err error) {
			r.Execution, err = deprecatedRepositories.TestResults().GetByNameAndTest(ctx, data.Name, data.TestName)
			return
		}),
		cloudresult.CmdResultGetLatestByTest: Handler(func(ctx context.Context, data cloudresult.GetLatestByTestRequest) (r cloudresult.GetLatestByTestResponse, err error) {
			ex, err := deprecatedRepositories.TestResults().GetLatestByTest(ctx, data.TestName)
			if ex != nil {
				r.Execution = *ex
			}
			return
		}),
		cloudresult.CmdResultGetLatestByTests: Handler(func(ctx context.Context, data cloudresult.GetLatestByTestsRequest) (r cloudresult.GetLatestByTestsResponse, err error) {
			r.Executions, err = deprecatedRepositories.TestResults().GetLatestByTests(ctx, data.TestNames)
			return
		}),
		cloudresult.CmdResultGetExecutionTotals: Handler(func(ctx context.Context, data cloudresult.GetExecutionTotalsRequest) (r cloudresult.GetExecutionTotalsResponse, err error) {
			r.Result, err = deprecatedRepositories.TestResults().GetExecutionTotals(ctx, data.Paging, mapTestFilters(data.Filter)...)
			return
		}),
		cloudresult.CmdResultGetExecutions: Handler(func(ctx context.Context, data cloudresult.GetExecutionsRequest) (r cloudresult.GetExecutionsResponse, err error) {
			r.Executions, err = deprecatedRepositories.TestResults().GetExecutions(ctx, data.Filter)
			return
		}),
		cloudresult.CmdResultGetPreviousFinishedState: Handler(func(ctx context.Context, data cloudresult.GetPreviousFinishedStateRequest) (r cloudresult.GetPreviousFinishedStateResponse, err error) {
			r.Result, err = deprecatedRepositories.TestResults().GetPreviousFinishedState(ctx, data.TestName, data.Date)
			return
		}),
		cloudresult.CmdResultInsert: Handler(func(ctx context.Context, data cloudresult.InsertRequest) (r cloudresult.InsertResponse, err error) {
			return r, deprecatedRepositories.TestResults().Insert(ctx, data.Result)
		}),
		cloudresult.CmdResultUpdate: Handler(func(ctx context.Context, data cloudresult.UpdateRequest) (r cloudresult.UpdateResponse, err error) {
			return r, deprecatedRepositories.TestResults().Update(ctx, data.Result)
		}),
		cloudresult.CmdResultUpdateResult: Handler(func(ctx context.Context, data cloudresult.UpdateResultInExecutionRequest) (r cloudresult.UpdateResultInExecutionResponse, err error) {
			return r, deprecatedRepositories.TestResults().UpdateResult(ctx, data.ID, data.Execution)
		}),
		cloudresult.CmdResultStartExecution: Handler(func(ctx context.Context, data cloudresult.StartExecutionRequest) (r cloudresult.StartExecutionResponse, err error) {
			return r, deprecatedRepositories.TestResults().StartExecution(ctx, data.ID, data.StartTime)
		}),
		cloudresult.CmdResultEndExecution: Handler(func(ctx context.Context, data cloudresult.EndExecutionRequest) (r cloudresult.EndExecutionResponse, err error) {
			return r, deprecatedRepositories.TestResults().EndExecution(ctx, data.Execution)
		}),
		cloudresult.CmdResultGetLabels: Handler(func(ctx context.Context, data cloudresult.GetLabelsRequest) (r cloudresult.GetLabelsResponse, err error) {
			r.Labels, err = deprecatedRepositories.TestResults().GetLabels(ctx)
			return
		}),
		cloudresult.CmdResultDeleteByTest: Handler(func(ctx context.Context, data cloudresult.DeleteByTestRequest) (r cloudresult.DeleteByTestResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteByTest(ctx, data.TestName)
		}),
		cloudresult.CmdResultDeleteByTestSuite: Handler(func(ctx context.Context, data cloudresult.DeleteByTestSuiteRequest) (r cloudresult.DeleteByTestSuiteResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteByTestSuite(ctx, data.TestSuiteName)
		}),
		cloudresult.CmdResultDeleteAll: Handler(func(ctx context.Context, data cloudresult.DeleteAllRequest) (r cloudresult.DeleteAllResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteAll(ctx)
		}),
		cloudresult.CmdResultDeleteByTests: Handler(func(ctx context.Context, data cloudresult.DeleteByTestsRequest) (r cloudresult.DeleteByTestsResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteByTests(ctx, data.TestNames)
		}),
		cloudresult.CmdResultDeleteByTestSuites: Handler(func(ctx context.Context, data cloudresult.DeleteByTestSuitesRequest) (r cloudresult.DeleteByTestSuitesResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteByTestSuites(ctx, data.TestSuiteNames)
		}),
		cloudresult.CmdResultDeleteForAllTestSuites: Handler(func(ctx context.Context, data cloudresult.DeleteForAllTestSuitesRequest) (r cloudresult.DeleteForAllTestSuitesResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteForAllTestSuites(ctx)
		}),
		cloudresult.CmdResultGetTestMetrics: Handler(func(ctx context.Context, data cloudresult.GetTestMetricsRequest) (r cloudresult.GetTestMetricsResponse, err error) {
			r.Metrics, err = deprecatedRepositories.TestResults().GetTestMetrics(ctx, data.Name, data.Limit, data.Last)
			return
		}),
		cloudresult.CmdResultGetNextExecutionNumber: Handler(func(ctx context.Context, data cloudresult.NextExecutionNumberRequest) (r cloudresult.NextExecutionNumberResponse, err error) {
			r.TestNumber, err = deprecatedRepositories.TestResults().GetNextExecutionNumber(ctx, data.TestName)
			return
		}),
	}

	// Set up "Test Suites - Executions" commands
	deprecatedTestSuiteExecutionsCommands := CommandHandlers{
		cloudtestresult.CmdTestResultGet: Handler(func(ctx context.Context, data cloudtestresult.GetRequest) (r cloudtestresult.GetResponse, err error) {
			r.TestSuiteExecution, err = deprecatedRepositories.TestSuiteResults().Get(ctx, data.ID)
			return
		}),
		cloudtestresult.CmdTestResultGetByNameAndTestSuite: Handler(func(ctx context.Context, data cloudtestresult.GetByNameAndTestSuiteRequest) (r cloudtestresult.GetByNameAndTestSuiteResponse, err error) {
			r.TestSuiteExecution, err = deprecatedRepositories.TestSuiteResults().GetByNameAndTestSuite(ctx, data.Name, data.TestSuiteName)
			return
		}),
		cloudtestresult.CmdTestResultGetLatestByTestSuite: Handler(func(ctx context.Context, data cloudtestresult.GetLatestByTestSuiteRequest) (r cloudtestresult.GetLatestByTestSuiteResponse, err error) {
			ex, err := deprecatedRepositories.TestSuiteResults().GetLatestByTestSuite(ctx, data.TestSuiteName)
			if ex != nil {
				r.TestSuiteExecution = *ex
			}
			return
		}),
		cloudtestresult.CmdTestResultGetLatestByTestSuites: Handler(func(ctx context.Context, data cloudtestresult.GetLatestByTestSuitesRequest) (r cloudtestresult.GetLatestByTestSuitesResponse, err error) {
			r.TestSuiteExecutions, err = deprecatedRepositories.TestSuiteResults().GetLatestByTestSuites(ctx, data.TestSuiteNames)
			return
		}),
		cloudtestresult.CmdTestResultGetExecutionsTotals: Handler(func(ctx context.Context, data cloudtestresult.GetExecutionsTotalsRequest) (r cloudtestresult.GetExecutionsTotalsResponse, err error) {
			r.ExecutionsTotals, err = deprecatedRepositories.TestSuiteResults().GetExecutionsTotals(ctx, mapTestSuiteFilters(data.Filter)...)
			return
		}),
		cloudtestresult.CmdTestResultGetExecutions: Handler(func(ctx context.Context, data cloudtestresult.GetExecutionsRequest) (r cloudtestresult.GetExecutionsResponse, err error) {
			r.TestSuiteExecutions, err = deprecatedRepositories.TestSuiteResults().GetExecutions(ctx, data.Filter)
			return
		}),
		cloudtestresult.CmdTestResultGetPreviousFinishedState: Handler(func(ctx context.Context, data cloudtestresult.GetPreviousFinishedStateRequest) (r cloudtestresult.GetPreviousFinishedStateResponse, err error) {
			r.Result, err = deprecatedRepositories.TestSuiteResults().GetPreviousFinishedState(ctx, data.TestSuiteName, data.Date)
			return
		}),
		cloudtestresult.CmdTestResultInsert: Handler(func(ctx context.Context, data cloudtestresult.InsertRequest) (r cloudtestresult.InsertResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().Insert(ctx, data.TestSuiteExecution)
		}),
		cloudtestresult.CmdTestResultUpdate: Handler(func(ctx context.Context, data cloudtestresult.UpdateRequest) (r cloudtestresult.UpdateResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().Update(ctx, data.TestSuiteExecution)
		}),
		cloudtestresult.CmdTestResultStartExecution: Handler(func(ctx context.Context, data cloudtestresult.StartExecutionRequest) (r cloudtestresult.StartExecutionResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().StartExecution(ctx, data.ID, data.StartTime)
		}),
		cloudtestresult.CmdTestResultEndExecution: Handler(func(ctx context.Context, data cloudtestresult.EndExecutionRequest) (r cloudtestresult.EndExecutionResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().EndExecution(ctx, data.Execution)
		}),
		cloudtestresult.CmdTestResultDeleteByTestSuite: Handler(func(ctx context.Context, data cloudtestresult.DeleteByTestSuiteRequest) (r cloudtestresult.DeleteByTestSuiteResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().DeleteByTestSuite(ctx, data.TestSuiteName)
		}),
		cloudtestresult.CmdTestResultDeleteAll: Handler(func(ctx context.Context, data cloudtestresult.DeleteAllTestResultsRequest) (r cloudtestresult.DeleteAllTestResultsResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().DeleteAll(ctx)
		}),
		cloudtestresult.CmdTestResultDeleteByTestSuites: Handler(func(ctx context.Context, data cloudtestresult.DeleteByTestSuitesRequest) (r cloudtestresult.DeleteByTestSuitesResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().DeleteByTestSuites(ctx, data.TestSuiteNames)
		}),
		cloudtestresult.CmdTestResultGetTestSuiteMetrics: Handler(func(ctx context.Context, data cloudtestresult.GetTestSuiteMetricsRequest) (r cloudtestresult.GetTestSuiteMetricsResponse, err error) {
			r.Metrics, err = deprecatedRepositories.TestSuiteResults().GetTestSuiteMetrics(ctx, data.Name, data.Limit, data.Last)
			return
		}),
		cloudtestresult.CmdTestResultGetNextExecutionNumber: Handler(func(ctx context.Context, data cloudtestresult.NextExecutionNumberRequest) (r cloudtestresult.NextExecutionNumberResponse, err error) {
			r.TestSuiteNumber, err = deprecatedRepositories.TestSuiteResults().GetNextExecutionNumber(ctx, data.TestSuiteName)
			return
		}),
	}

	// Set up "Test Workflows - Executions" commands
	testWorkflowExecutionsCommands := CommandHandlers{
		cloudtestworkflow.CmdTestWorkflowExecutionGet: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetRequest) (r cloudtestworkflow.ExecutionGetResponse, err error) {
			r.WorkflowExecution, err = testWorkflowResultsRepository.Get(ctx, data.ID)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetByNameAndWorkflow: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetByNameAndWorkflowRequest) (r cloudtestworkflow.ExecutionGetByNameAndWorkflowResponse, err error) {
			r.WorkflowExecution, err = testWorkflowResultsRepository.GetByNameAndTestWorkflow(ctx, data.Name, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetLatestByWorkflow: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetLatestByWorkflowRequest) (r cloudtestworkflow.ExecutionGetLatestByWorkflowResponse, err error) {
			sortBy := testworkflow.ParseLatestSortBy(data.SortBy)
			r.WorkflowExecution, err = testWorkflowResultsRepository.GetLatestByTestWorkflow(ctx, data.WorkflowName, sortBy)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetRunning: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetRunningRequest) (r cloudtestworkflow.ExecutionGetRunningResponse, err error) {
			r.WorkflowExecutions, err = testWorkflowResultsRepository.GetRunning(ctx)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetLatestByWorkflows: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetLatestByWorkflowsRequest) (r cloudtestworkflow.ExecutionGetLatestByWorkflowsResponse, err error) {
			r.WorkflowExecutions, err = testWorkflowResultsRepository.GetLatestByTestWorkflows(ctx, data.WorkflowNames)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetExecutionTotals: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetExecutionTotalsRequest) (r cloudtestworkflow.ExecutionGetExecutionTotalsResponse, err error) {
			r.Totals, err = testWorkflowResultsRepository.GetExecutionsTotals(ctx, mapTestWorkflowFilters(data.Filter)...)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetExecutions: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetExecutionsRequest) (r cloudtestworkflow.ExecutionGetExecutionsResponse, err error) {
			r.WorkflowExecutions, err = testWorkflowResultsRepository.GetExecutions(ctx, data.Filter)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetExecutionsSummary: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetExecutionsSummaryRequest) (r cloudtestworkflow.ExecutionGetExecutionsSummaryResponse, err error) {
			r.WorkflowExecutions, err = testWorkflowResultsRepository.GetExecutionsSummary(ctx, data.Filter)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetPreviousFinishedState: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetPreviousFinishedStateRequest) (r cloudtestworkflow.ExecutionGetPreviousFinishedStateResponse, err error) {
			r.Result, err = testWorkflowResultsRepository.GetPreviousFinishedState(ctx, data.WorkflowName, data.Date)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionInsert: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionInsertRequest) (r cloudtestworkflow.ExecutionInsertResponse, err error) {
			return r, testWorkflowResultsRepository.Insert(ctx, data.WorkflowExecution)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionUpdate: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionUpdateRequest) (r cloudtestworkflow.ExecutionUpdateResponse, err error) {
			return r, testWorkflowResultsRepository.Update(ctx, data.WorkflowExecution)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionUpdateResult: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionUpdateResultRequest) (r cloudtestworkflow.ExecutionUpdateResultResponse, err error) {
			return r, testWorkflowResultsRepository.UpdateResult(ctx, data.ID, data.Result)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionUpdateOutput: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionUpdateOutputRequest) (r cloudtestworkflow.ExecutionUpdateOutputResponse, err error) {
			return r, testWorkflowResultsRepository.UpdateOutput(ctx, data.ID, data.Output)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionDeleteByWorkflow: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteByWorkflowRequest) (r cloudtestworkflow.ExecutionDeleteByWorkflowResponse, err error) {
			return r, testWorkflowResultsRepository.DeleteByTestWorkflow(ctx, data.WorkflowName)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionDeleteAll: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteAllRequest) (r cloudtestworkflow.ExecutionDeleteAllResponse, err error) {
			return r, testWorkflowResultsRepository.DeleteAll(ctx)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionDeleteByWorkflows: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteByWorkflowsRequest) (r cloudtestworkflow.ExecutionDeleteByWorkflowsResponse, err error) {
			return r, testWorkflowResultsRepository.DeleteByTestWorkflows(ctx, data.WorkflowNames)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetWorkflowMetrics: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetWorkflowMetricsRequest) (r cloudtestworkflow.ExecutionGetWorkflowMetricsResponse, err error) {
			r.Metrics, err = testWorkflowResultsRepository.GetTestWorkflowMetrics(ctx, data.Name, data.Limit, data.Last)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionAddReport: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionsAddReportRequest) (r cloudtestworkflow.ExecutionsAddReportResponse, err error) {
			return r, status.Error(codes.Unimplemented, "not supported in the standalone mode")
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetNextExecutionNumber: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetNextExecutionNumberRequest) (r cloudtestworkflow.ExecutionGetNextExecutionNumberResponse, err error) {
			r.TestWorkflowNumber, err = testWorkflowResultsRepository.GetNextExecutionNumber(ctx, data.TestWorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetExecutionTags: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetExecutionTagsRequest) (r cloudtestworkflow.ExecutionGetExecutionTagsResponse, err error) {
			r.Tags, err = testWorkflowResultsRepository.GetExecutionTags(ctx, data.TestWorkflowName)
			return
		}),
	}

	// Set up "Test Workflows - Output" commands
	testWorkflowsOutputCommands := CommandHandlers{
		cloudtestworkflow.CmdTestWorkflowOutputPresignSaveLog: Handler(func(ctx context.Context, data cloudtestworkflow.OutputPresignSaveLogRequest) (r cloudtestworkflow.OutputPresignSaveLogResponse, err error) {
			r.URL, err = testWorkflowOutputRepository.PresignSaveLog(ctx, data.ID, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowOutputPresignReadLog: Handler(func(ctx context.Context, data cloudtestworkflow.OutputPresignReadLogRequest) (r cloudtestworkflow.OutputPresignReadLogResponse, err error) {
			r.URL, err = testWorkflowOutputRepository.PresignReadLog(ctx, data.ID, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowOutputHasLog: Handler(func(ctx context.Context, data cloudtestworkflow.OutputHasLogRequest) (r cloudtestworkflow.OutputHasLogResponse, err error) {
			r.Has, err = testWorkflowOutputRepository.HasLog(ctx, data.ID, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowOutputDeleteByTestWorkflow: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteOutputByWorkflowRequest) (r cloudtestworkflow.ExecutionDeleteOutputByWorkflowResponse, err error) {
			return r, testWorkflowOutputRepository.DeleteOutputByTestWorkflow(ctx, data.WorkflowName)
		}),
		cloudtestworkflow.CmdTestworkflowOutputDeleteForTestWorkflows: Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteOutputForTestWorkflowsRequest) (r cloudtestworkflow.ExecutionDeleteOutputForTestWorkflowsResponse, err error) {
			return r, testWorkflowOutputRepository.DeleteOutputForTestWorkflows(ctx, data.WorkflowNames)
		}),
	}

	// Set up "Artifacts" commands
	// TODO: What about downloading artifacts archive?
	// TODO: What about handling ArtifactRequest.OmitFolderPerExecution and ArtifactRequest.StorageBucket?
	artifactsCommands := CommandHandlers{
		cloudartifacts.CmdScraperPutObjectSignedURL: Handler(func(ctx context.Context, data cloudartifacts.PutObjectSignedURLRequest) (r cloudartifacts.PutObjectSignedURLResponse, err error) {
			r.URL, err = storageClient.PresignUploadFileToBucket(ctx, storageBucket, data.ExecutionID, data.Object, 15*time.Minute)
			return r, err
		}),
		cloudartifacts.CmdArtifactsListFiles: Handler(func(ctx context.Context, data cloudartifacts.ListFilesRequest) (r cloudartifacts.ListFilesResponse, err error) {
			r.Artifacts, err = artifactStorage.ListFiles(ctx, data.ExecutionID, data.TestName, data.TestSuiteName, data.TestWorkflowName)
			return r, err
		}),
		cloudartifacts.CmdArtifactsDownloadFile: Handler(func(ctx context.Context, data cloudartifacts.DownloadFileRequest) (r cloudartifacts.DownloadFileResponse, err error) {
			r.URL, err = storageClient.PresignDownloadFileFromBucket(ctx, storageBucket, data.ExecutionID, data.File, 15*time.Minute)
			return r, err
		}),
	}

	// Select commands to use
	commands := []CommandHandlers{configCommands, testWorkflowExecutionsCommands, testWorkflowsOutputCommands, artifactsCommands, webhoookCommands}
	if !disableDeprecatedTests {
		commands = append(commands, deprecatedTestExecutionsCommands, deprecatedTestSuiteExecutionsCommands)
	}
	return commands
}

func mapTestWorkflowFilters(s []*testworkflow.FilterImpl) []testworkflow.Filter {
	v := make([]testworkflow.Filter, len(s))
	for i := range s {
		v[i] = s[i]
	}
	return v
}

func mapTestFilters(s []*result.FilterImpl) []result.Filter {
	v := make([]result.Filter, len(s))
	for i := range s {
		v[i] = s[i]
	}
	return v
}

func mapTestSuiteFilters(s []*testresult.FilterImpl) []testresult.Filter {
	v := make([]testresult.Filter, len(s))
	for i := range s {
		v[i] = s[i]
	}
	return v
}
