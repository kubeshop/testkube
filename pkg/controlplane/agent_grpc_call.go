package controlplane

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/cloud"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudconfig "github.com/kubeshop/testkube/pkg/cloud/data/config"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
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

type CommandNotImplementedError string

func (e CommandNotImplementedError) Error() string {
	return fmt.Sprintf("command not implemented: %s", string(e))
}

func (s *Server) Call(ctx context.Context, request *cloud.CommandRequest) (*cloud.CommandResponse, error) {
	if cmd, ok := s.commands[cloudexecutor.Command(request.Command)]; ok {
		return cmd(ctx, request)
	}
	return nil, CommandNotImplementedError(request.Command)
}

func CreateCommands(storageBucket string, storageClient domainstorage.Client, testWorkflowOutputRepository *miniorepo.MinioRepository, testWorkflowResultsRepository testworkflow.Repository, artifactStorage *minio.ArtifactClient) []CommandHandlers {
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

	return []CommandHandlers{configCommands, testWorkflowExecutionsCommands, testWorkflowsOutputCommands, artifactsCommands, webhoookCommands}
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
