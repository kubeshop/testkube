package services

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"

	kubeclient "github.com/kubeshop/testkube-operator/pkg/client"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudconfig "github.com/kubeshop/testkube/pkg/cloud/data/config"
	cloudresult "github.com/kubeshop/testkube/pkg/cloud/data/result"
	cloudtestresult "github.com/kubeshop/testkube/pkg/cloud/data/testresult"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/controlplane"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository"
	"github.com/kubeshop/testkube/pkg/repository/result"
	minioresult "github.com/kubeshop/testkube/pkg/repository/result/minio"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	miniorepo "github.com/kubeshop/testkube/pkg/repository/testworkflow/minio"
	runner2 "github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

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

func CreateControlPlane(ctx context.Context, cfg *config.Config, features featureflags.FeatureFlags, secretManager secretmanager.SecretManager, metrics metrics.Metrics, runner runner2.RunnerExecute, emitter event.Interface) *controlplane.Server {
	// Connect to the cluster
	kubeConfig, err := k8sclient.GetK8sClientConfig()
	commons.ExitOnError("Getting kubernetes config", err)
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	commons.ExitOnError("Creating k8s clientset", err)
	kubeClient, err := kubeclient.GetClient()
	commons.ExitOnError("Getting kubernetes client", err)

	// Connect to storages
	secretClient := secret.NewClientFor(clientset, cfg.TestkubeNamespace)
	storageClient := commons.MustGetMinioClient(cfg)

	var logGrpcClient logsclient.StreamGetter
	if !cfg.DisableDeprecatedTests && features.LogsV2 {
		logGrpcClient = commons.MustGetLogsV2Client(cfg)
		commons.ExitOnError("Creating logs streaming client", err)
	}

	var factory repository.RepositoryFactory
	if cfg.APIMongoDSN != "" {
		mongoDb := commons.MustGetMongoDatabase(ctx, cfg, secretClient, !cfg.DisableMongoMigrations)
		factory, err = CreateMongoFactory(ctx, cfg, mongoDb, logGrpcClient, storageClient, features)
	}
	if cfg.APIPostgresDSN != "" {
		postgresDb := commons.MustGetPostgresDatabase(ctx, cfg)
		factory, err = CreatePostgresFactory(postgresDb)
	}
	commons.ExitOnError("Creating factory for database", err)

	testWorkflowsClient, err := testworkflowclient.NewKubernetesTestWorkflowClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
	commons.ExitOnError("Creating test workflow client", err)
	testWorkflowTemplatesClient, err := testworkflowtemplateclient.NewKubernetesTestWorkflowTemplateClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
	commons.ExitOnError("Creating test workflow templates client", err)

	// Build repositories
	repoManager := repository.NewRepositoryManager(factory)
	testWorkflowResultsRepository := repoManager.TestWorkflow()
	testWorkflowOutputRepository := miniorepo.NewMinioOutputRepository(storageClient, testWorkflowResultsRepository, cfg.LogsBucket)
	deprecatedRepositories := commons.CreateDeprecatedRepositoriesForMongo(repoManager)
	artifactStorage := minio.NewMinIOArtifactClient(storageClient)

	// Set up "Config" commands
	configCommands := controlplane.CommandHandlers{
		cloudconfig.CmdConfigGetOrganizationPlan: controlplane.Handler(func(ctx context.Context, data checktcl.GetOrganizationPlanRequest) (r checktcl.GetOrganizationPlanResponse, err error) {
			return
		}),
	}

	// Set up "Webhook commands
	webhoookCommands := controlplane.CommandHandlers{
		cloudwebhook.CmdWebhookExecutionCollectResult: controlplane.Handler(func(ctx context.Context, data cloudwebhook.WebhookExecutionCollectResultRequest) (r cloudwebhook.WebhookExecutionCollectResultResponse, err error) {
			return
		}),
	}

	// Set up "Tests - Executions" commands
	deprecatedTestExecutionsCommands := controlplane.CommandHandlers{
		cloudresult.CmdResultGet: controlplane.Handler(func(ctx context.Context, data cloudresult.GetRequest) (r cloudresult.GetResponse, err error) {
			r.Execution, err = deprecatedRepositories.TestResults().Get(ctx, data.ID)
			return
		}),
		cloudresult.CmdResultGetByNameAndTest: controlplane.Handler(func(ctx context.Context, data cloudresult.GetByNameAndTestRequest) (r cloudresult.GetByNameAndTestResponse, err error) {
			r.Execution, err = deprecatedRepositories.TestResults().GetByNameAndTest(ctx, data.Name, data.TestName)
			return
		}),
		cloudresult.CmdResultGetLatestByTest: controlplane.Handler(func(ctx context.Context, data cloudresult.GetLatestByTestRequest) (r cloudresult.GetLatestByTestResponse, err error) {
			ex, err := deprecatedRepositories.TestResults().GetLatestByTest(ctx, data.TestName)
			if ex != nil {
				r.Execution = *ex
			}
			return
		}),
		cloudresult.CmdResultGetLatestByTests: controlplane.Handler(func(ctx context.Context, data cloudresult.GetLatestByTestsRequest) (r cloudresult.GetLatestByTestsResponse, err error) {
			r.Executions, err = deprecatedRepositories.TestResults().GetLatestByTests(ctx, data.TestNames)
			return
		}),
		cloudresult.CmdResultGetExecutionTotals: controlplane.Handler(func(ctx context.Context, data cloudresult.GetExecutionTotalsRequest) (r cloudresult.GetExecutionTotalsResponse, err error) {
			r.Result, err = deprecatedRepositories.TestResults().GetExecutionTotals(ctx, data.Paging, mapTestFilters(data.Filter)...)
			return
		}),
		cloudresult.CmdResultGetExecutions: controlplane.Handler(func(ctx context.Context, data cloudresult.GetExecutionsRequest) (r cloudresult.GetExecutionsResponse, err error) {
			r.Executions, err = deprecatedRepositories.TestResults().GetExecutions(ctx, data.Filter)
			return
		}),
		cloudresult.CmdResultGetPreviousFinishedState: controlplane.Handler(func(ctx context.Context, data cloudresult.GetPreviousFinishedStateRequest) (r cloudresult.GetPreviousFinishedStateResponse, err error) {
			r.Result, err = deprecatedRepositories.TestResults().GetPreviousFinishedState(ctx, data.TestName, data.Date)
			return
		}),
		cloudresult.CmdResultInsert: controlplane.Handler(func(ctx context.Context, data cloudresult.InsertRequest) (r cloudresult.InsertResponse, err error) {
			return r, deprecatedRepositories.TestResults().Insert(ctx, data.Result)
		}),
		cloudresult.CmdResultUpdate: controlplane.Handler(func(ctx context.Context, data cloudresult.UpdateRequest) (r cloudresult.UpdateResponse, err error) {
			return r, deprecatedRepositories.TestResults().Update(ctx, data.Result)
		}),
		cloudresult.CmdResultUpdateResult: controlplane.Handler(func(ctx context.Context, data cloudresult.UpdateResultInExecutionRequest) (r cloudresult.UpdateResultInExecutionResponse, err error) {
			return r, deprecatedRepositories.TestResults().UpdateResult(ctx, data.ID, data.Execution)
		}),
		cloudresult.CmdResultStartExecution: controlplane.Handler(func(ctx context.Context, data cloudresult.StartExecutionRequest) (r cloudresult.StartExecutionResponse, err error) {
			return r, deprecatedRepositories.TestResults().StartExecution(ctx, data.ID, data.StartTime)
		}),
		cloudresult.CmdResultEndExecution: controlplane.Handler(func(ctx context.Context, data cloudresult.EndExecutionRequest) (r cloudresult.EndExecutionResponse, err error) {
			return r, deprecatedRepositories.TestResults().EndExecution(ctx, data.Execution)
		}),
		cloudresult.CmdResultGetLabels: controlplane.Handler(func(ctx context.Context, data cloudresult.GetLabelsRequest) (r cloudresult.GetLabelsResponse, err error) {
			r.Labels, err = deprecatedRepositories.TestResults().GetLabels(ctx)
			return
		}),
		cloudresult.CmdResultDeleteByTest: controlplane.Handler(func(ctx context.Context, data cloudresult.DeleteByTestRequest) (r cloudresult.DeleteByTestResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteByTest(ctx, data.TestName)
		}),
		cloudresult.CmdResultDeleteByTestSuite: controlplane.Handler(func(ctx context.Context, data cloudresult.DeleteByTestSuiteRequest) (r cloudresult.DeleteByTestSuiteResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteByTestSuite(ctx, data.TestSuiteName)
		}),
		cloudresult.CmdResultDeleteAll: controlplane.Handler(func(ctx context.Context, data cloudresult.DeleteAllRequest) (r cloudresult.DeleteAllResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteAll(ctx)
		}),
		cloudresult.CmdResultDeleteByTests: controlplane.Handler(func(ctx context.Context, data cloudresult.DeleteByTestsRequest) (r cloudresult.DeleteByTestsResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteByTests(ctx, data.TestNames)
		}),
		cloudresult.CmdResultDeleteByTestSuites: controlplane.Handler(func(ctx context.Context, data cloudresult.DeleteByTestSuitesRequest) (r cloudresult.DeleteByTestSuitesResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteByTestSuites(ctx, data.TestSuiteNames)
		}),
		cloudresult.CmdResultDeleteForAllTestSuites: controlplane.Handler(func(ctx context.Context, data cloudresult.DeleteForAllTestSuitesRequest) (r cloudresult.DeleteForAllTestSuitesResponse, err error) {
			return r, deprecatedRepositories.TestResults().DeleteForAllTestSuites(ctx)
		}),
		cloudresult.CmdResultGetTestMetrics: controlplane.Handler(func(ctx context.Context, data cloudresult.GetTestMetricsRequest) (r cloudresult.GetTestMetricsResponse, err error) {
			r.Metrics, err = deprecatedRepositories.TestResults().GetTestMetrics(ctx, data.Name, data.Limit, data.Last)
			return
		}),
		cloudresult.CmdResultGetNextExecutionNumber: controlplane.Handler(func(ctx context.Context, data cloudresult.NextExecutionNumberRequest) (r cloudresult.NextExecutionNumberResponse, err error) {
			r.TestNumber, err = deprecatedRepositories.TestResults().GetNextExecutionNumber(ctx, data.TestName)
			return
		}),
	}

	// Set up "Test Suites - Executions" commands
	deprecatedTestSuiteExecutionsCommands := controlplane.CommandHandlers{
		cloudtestresult.CmdTestResultGet: controlplane.Handler(func(ctx context.Context, data cloudtestresult.GetRequest) (r cloudtestresult.GetResponse, err error) {
			r.TestSuiteExecution, err = deprecatedRepositories.TestSuiteResults().Get(ctx, data.ID)
			return
		}),
		cloudtestresult.CmdTestResultGetByNameAndTestSuite: controlplane.Handler(func(ctx context.Context, data cloudtestresult.GetByNameAndTestSuiteRequest) (r cloudtestresult.GetByNameAndTestSuiteResponse, err error) {
			r.TestSuiteExecution, err = deprecatedRepositories.TestSuiteResults().GetByNameAndTestSuite(ctx, data.Name, data.TestSuiteName)
			return
		}),
		cloudtestresult.CmdTestResultGetLatestByTestSuite: controlplane.Handler(func(ctx context.Context, data cloudtestresult.GetLatestByTestSuiteRequest) (r cloudtestresult.GetLatestByTestSuiteResponse, err error) {
			ex, err := deprecatedRepositories.TestSuiteResults().GetLatestByTestSuite(ctx, data.TestSuiteName)
			if ex != nil {
				r.TestSuiteExecution = *ex
			}
			return
		}),
		cloudtestresult.CmdTestResultGetLatestByTestSuites: controlplane.Handler(func(ctx context.Context, data cloudtestresult.GetLatestByTestSuitesRequest) (r cloudtestresult.GetLatestByTestSuitesResponse, err error) {
			r.TestSuiteExecutions, err = deprecatedRepositories.TestSuiteResults().GetLatestByTestSuites(ctx, data.TestSuiteNames)
			return
		}),
		cloudtestresult.CmdTestResultGetExecutionsTotals: controlplane.Handler(func(ctx context.Context, data cloudtestresult.GetExecutionsTotalsRequest) (r cloudtestresult.GetExecutionsTotalsResponse, err error) {
			r.ExecutionsTotals, err = deprecatedRepositories.TestSuiteResults().GetExecutionsTotals(ctx, mapTestSuiteFilters(data.Filter)...)
			return
		}),
		cloudtestresult.CmdTestResultGetExecutions: controlplane.Handler(func(ctx context.Context, data cloudtestresult.GetExecutionsRequest) (r cloudtestresult.GetExecutionsResponse, err error) {
			r.TestSuiteExecutions, err = deprecatedRepositories.TestSuiteResults().GetExecutions(ctx, data.Filter)
			return
		}),
		cloudtestresult.CmdTestResultGetPreviousFinishedState: controlplane.Handler(func(ctx context.Context, data cloudtestresult.GetPreviousFinishedStateRequest) (r cloudtestresult.GetPreviousFinishedStateResponse, err error) {
			r.Result, err = deprecatedRepositories.TestSuiteResults().GetPreviousFinishedState(ctx, data.TestSuiteName, data.Date)
			return
		}),
		cloudtestresult.CmdTestResultInsert: controlplane.Handler(func(ctx context.Context, data cloudtestresult.InsertRequest) (r cloudtestresult.InsertResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().Insert(ctx, data.TestSuiteExecution)
		}),
		cloudtestresult.CmdTestResultUpdate: controlplane.Handler(func(ctx context.Context, data cloudtestresult.UpdateRequest) (r cloudtestresult.UpdateResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().Update(ctx, data.TestSuiteExecution)
		}),
		cloudtestresult.CmdTestResultStartExecution: controlplane.Handler(func(ctx context.Context, data cloudtestresult.StartExecutionRequest) (r cloudtestresult.StartExecutionResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().StartExecution(ctx, data.ID, data.StartTime)
		}),
		cloudtestresult.CmdTestResultEndExecution: controlplane.Handler(func(ctx context.Context, data cloudtestresult.EndExecutionRequest) (r cloudtestresult.EndExecutionResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().EndExecution(ctx, data.Execution)
		}),
		cloudtestresult.CmdTestResultDeleteByTestSuite: controlplane.Handler(func(ctx context.Context, data cloudtestresult.DeleteByTestSuiteRequest) (r cloudtestresult.DeleteByTestSuiteResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().DeleteByTestSuite(ctx, data.TestSuiteName)
		}),
		cloudtestresult.CmdTestResultDeleteAll: controlplane.Handler(func(ctx context.Context, data cloudtestresult.DeleteAllTestResultsRequest) (r cloudtestresult.DeleteAllTestResultsResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().DeleteAll(ctx)
		}),
		cloudtestresult.CmdTestResultDeleteByTestSuites: controlplane.Handler(func(ctx context.Context, data cloudtestresult.DeleteByTestSuitesRequest) (r cloudtestresult.DeleteByTestSuitesResponse, err error) {
			return r, deprecatedRepositories.TestSuiteResults().DeleteByTestSuites(ctx, data.TestSuiteNames)
		}),
		cloudtestresult.CmdTestResultGetTestSuiteMetrics: controlplane.Handler(func(ctx context.Context, data cloudtestresult.GetTestSuiteMetricsRequest) (r cloudtestresult.GetTestSuiteMetricsResponse, err error) {
			r.Metrics, err = deprecatedRepositories.TestSuiteResults().GetTestSuiteMetrics(ctx, data.Name, data.Limit, data.Last)
			return
		}),
		cloudtestresult.CmdTestResultGetNextExecutionNumber: controlplane.Handler(func(ctx context.Context, data cloudtestresult.NextExecutionNumberRequest) (r cloudtestresult.NextExecutionNumberResponse, err error) {
			r.TestSuiteNumber, err = deprecatedRepositories.TestSuiteResults().GetNextExecutionNumber(ctx, data.TestSuiteName)
			return
		}),
	}

	// Set up "Test Workflows - Executions" commands
	testWorkflowExecutionsCommands := controlplane.CommandHandlers{
		cloudtestworkflow.CmdTestWorkflowExecutionGet: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetRequest) (r cloudtestworkflow.ExecutionGetResponse, err error) {
			r.WorkflowExecution, err = testWorkflowResultsRepository.Get(ctx, data.ID)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetByNameAndWorkflow: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetByNameAndWorkflowRequest) (r cloudtestworkflow.ExecutionGetByNameAndWorkflowResponse, err error) {
			r.WorkflowExecution, err = testWorkflowResultsRepository.GetByNameAndTestWorkflow(ctx, data.Name, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetLatestByWorkflow: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetLatestByWorkflowRequest) (r cloudtestworkflow.ExecutionGetLatestByWorkflowResponse, err error) {
			r.WorkflowExecution, err = testWorkflowResultsRepository.GetLatestByTestWorkflow(ctx, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetRunning: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetRunningRequest) (r cloudtestworkflow.ExecutionGetRunningResponse, err error) {
			r.WorkflowExecutions, err = testWorkflowResultsRepository.GetRunning(ctx)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetLatestByWorkflows: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetLatestByWorkflowsRequest) (r cloudtestworkflow.ExecutionGetLatestByWorkflowsResponse, err error) {
			r.WorkflowExecutions, err = testWorkflowResultsRepository.GetLatestByTestWorkflows(ctx, data.WorkflowNames)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetExecutionTotals: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetExecutionTotalsRequest) (r cloudtestworkflow.ExecutionGetExecutionTotalsResponse, err error) {
			r.Totals, err = testWorkflowResultsRepository.GetExecutionsTotals(ctx, mapTestWorkflowFilters(data.Filter)...)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetExecutions: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetExecutionsRequest) (r cloudtestworkflow.ExecutionGetExecutionsResponse, err error) {
			r.WorkflowExecutions, err = testWorkflowResultsRepository.GetExecutions(ctx, data.Filter)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetExecutionsSummary: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetExecutionsSummaryRequest) (r cloudtestworkflow.ExecutionGetExecutionsSummaryResponse, err error) {
			r.WorkflowExecutions, err = testWorkflowResultsRepository.GetExecutionsSummary(ctx, data.Filter)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetPreviousFinishedState: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetPreviousFinishedStateRequest) (r cloudtestworkflow.ExecutionGetPreviousFinishedStateResponse, err error) {
			r.Result, err = testWorkflowResultsRepository.GetPreviousFinishedState(ctx, data.WorkflowName, data.Date)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionInsert: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionInsertRequest) (r cloudtestworkflow.ExecutionInsertResponse, err error) {
			return r, testWorkflowResultsRepository.Insert(ctx, data.WorkflowExecution)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionUpdate: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionUpdateRequest) (r cloudtestworkflow.ExecutionUpdateResponse, err error) {
			return r, testWorkflowResultsRepository.Update(ctx, data.WorkflowExecution)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionUpdateResult: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionUpdateResultRequest) (r cloudtestworkflow.ExecutionUpdateResultResponse, err error) {
			return r, testWorkflowResultsRepository.UpdateResult(ctx, data.ID, data.Result)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionUpdateOutput: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionUpdateOutputRequest) (r cloudtestworkflow.ExecutionUpdateOutputResponse, err error) {
			return r, testWorkflowResultsRepository.UpdateOutput(ctx, data.ID, data.Output)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionDeleteByWorkflow: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteByWorkflowRequest) (r cloudtestworkflow.ExecutionDeleteByWorkflowResponse, err error) {
			return r, testWorkflowResultsRepository.DeleteByTestWorkflow(ctx, data.WorkflowName)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionDeleteAll: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteAllRequest) (r cloudtestworkflow.ExecutionDeleteAllResponse, err error) {
			return r, testWorkflowResultsRepository.DeleteAll(ctx)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionDeleteByWorkflows: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteByWorkflowsRequest) (r cloudtestworkflow.ExecutionDeleteByWorkflowsResponse, err error) {
			return r, testWorkflowResultsRepository.DeleteByTestWorkflows(ctx, data.WorkflowNames)
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetWorkflowMetrics: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetWorkflowMetricsRequest) (r cloudtestworkflow.ExecutionGetWorkflowMetricsResponse, err error) {
			r.Metrics, err = testWorkflowResultsRepository.GetTestWorkflowMetrics(ctx, data.Name, data.Limit, data.Last)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionAddReport: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionsAddReportRequest) (r cloudtestworkflow.ExecutionsAddReportResponse, err error) {
			return r, status.Error(codes.Unimplemented, "not supported in the standalone mode")
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetNextExecutionNumber: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetNextExecutionNumberRequest) (r cloudtestworkflow.ExecutionGetNextExecutionNumberResponse, err error) {
			r.TestWorkflowNumber, err = testWorkflowResultsRepository.GetNextExecutionNumber(ctx, data.TestWorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowExecutionGetExecutionTags: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionGetExecutionTagsRequest) (r cloudtestworkflow.ExecutionGetExecutionTagsResponse, err error) {
			r.Tags, err = testWorkflowResultsRepository.GetExecutionTags(ctx, data.TestWorkflowName)
			return
		}),
	}

	// Set up "Test Workflows - Output" commands
	testWorkflowsOutputCommands := controlplane.CommandHandlers{
		cloudtestworkflow.CmdTestWorkflowOutputPresignSaveLog: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.OutputPresignSaveLogRequest) (r cloudtestworkflow.OutputPresignSaveLogResponse, err error) {
			r.URL, err = testWorkflowOutputRepository.PresignSaveLog(ctx, data.ID, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowOutputPresignReadLog: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.OutputPresignReadLogRequest) (r cloudtestworkflow.OutputPresignReadLogResponse, err error) {
			r.URL, err = testWorkflowOutputRepository.PresignReadLog(ctx, data.ID, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowOutputHasLog: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.OutputHasLogRequest) (r cloudtestworkflow.OutputHasLogResponse, err error) {
			r.Has, err = testWorkflowOutputRepository.HasLog(ctx, data.ID, data.WorkflowName)
			return
		}),
		cloudtestworkflow.CmdTestWorkflowOutputDeleteByTestWorkflow: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteOutputByWorkflowRequest) (r cloudtestworkflow.ExecutionDeleteOutputByWorkflowResponse, err error) {
			return r, testWorkflowOutputRepository.DeleteOutputByTestWorkflow(ctx, data.WorkflowName)
		}),
		cloudtestworkflow.CmdTestworkflowOutputDeleteForTestWorkflows: controlplane.Handler(func(ctx context.Context, data cloudtestworkflow.ExecutionDeleteOutputForTestWorkflowsRequest) (r cloudtestworkflow.ExecutionDeleteOutputForTestWorkflowsResponse, err error) {
			return r, testWorkflowOutputRepository.DeleteOutputForTestWorkflows(ctx, data.WorkflowNames)
		}),
	}

	// Set up "Artifacts" commands
	// TODO: What about downloading artifacts archive?
	// TODO: What about handling ArtifactRequest.OmitFolderPerExecution and ArtifactRequest.StorageBucket?
	artifactsCommands := controlplane.CommandHandlers{
		cloudartifacts.CmdScraperPutObjectSignedURL: controlplane.Handler(func(ctx context.Context, data cloudartifacts.PutObjectSignedURLRequest) (r cloudartifacts.PutObjectSignedURLResponse, err error) {
			r.URL, err = storageClient.PresignUploadFileToBucket(ctx, cfg.StorageBucket, data.ExecutionID, data.Object, 15*time.Minute)
			return r, err
		}),
		cloudartifacts.CmdArtifactsListFiles: controlplane.Handler(func(ctx context.Context, data cloudartifacts.ListFilesRequest) (r cloudartifacts.ListFilesResponse, err error) {
			r.Artifacts, err = artifactStorage.ListFiles(ctx, data.ExecutionID, data.TestName, data.TestSuiteName, data.TestWorkflowName)
			return r, err
		}),
		cloudartifacts.CmdArtifactsDownloadFile: controlplane.Handler(func(ctx context.Context, data cloudartifacts.DownloadFileRequest) (r cloudartifacts.DownloadFileResponse, err error) {
			r.URL, err = storageClient.PresignDownloadFileFromBucket(ctx, cfg.StorageBucket, data.ExecutionID, data.File, 15*time.Minute)
			return r, err
		}),
	}

	// Select commands to use
	commands := []controlplane.CommandHandlers{configCommands, testWorkflowExecutionsCommands, testWorkflowsOutputCommands, artifactsCommands, webhoookCommands}
	if !cfg.DisableDeprecatedTests {
		commands = append(commands, deprecatedTestExecutionsCommands, deprecatedTestSuiteExecutionsCommands)
	}

	// Ensure the buckets exist
	if cfg.StorageBucket != "" {
		exists, err := storageClient.BucketExists(ctx, cfg.StorageBucket)
		if err != nil {
			log.DefaultLogger.Errorw("Failed to check if the storage bucket exists", "error", err)
		} else if !exists {
			err = storageClient.CreateBucket(ctx, cfg.StorageBucket)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				log.DefaultLogger.Errorw("Creating storage bucket", "error", err)
			}
		}
	}
	if cfg.LogsBucket != "" {
		exists, err := storageClient.BucketExists(ctx, cfg.LogsBucket)
		if err != nil {
			log.DefaultLogger.Errorw("Failed to check if the storage bucket exists", "error", err)
		} else if !exists {
			err = storageClient.CreateBucket(ctx, cfg.LogsBucket)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				log.DefaultLogger.Errorw("Creating logs bucket", "error", err)
			}
		}
	}

	executor := testworkflowexecutor.New(
		nil,
		"",
		cfg.CDEventsTarget,
		emitter,
		runner,
		testWorkflowResultsRepository,
		testWorkflowOutputRepository,
		testWorkflowTemplatesClient,
		testWorkflowsClient,
		metrics,
		secretManager,
		cfg.GlobalWorkflowTemplateName,
		cfg.TestkubeDashboardURI,
		"",
		"",
		"",
		nil,
		"",
		cfg.FeatureNewArchitecture,
	)

	return controlplane.New(controlplane.Config{
		Port:                             cfg.GRPCServerPort,
		Logger:                           log.DefaultLogger,
		Verbose:                          false,
		StorageBucket:                    cfg.StorageBucket,
		FeatureNewArchitecture:           cfg.FeatureNewArchitecture,
		FeatureTestWorkflowsCloudStorage: cfg.FeatureCloudStorage,
	}, executor, storageClient, testWorkflowsClient, testWorkflowTemplatesClient,
		testWorkflowResultsRepository, testWorkflowOutputRepository, repoManager, commands...)
}

func CreateMongoFactory(ctx context.Context, cfg *config.Config, db *mongo.Database,
	logGrpcClient logsclient.StreamGetter, storageClient domainstorage.Client, features featureflags.FeatureFlags) (repository.RepositoryFactory, error) {
	var outputRepository *minioresult.MinioRepository
	// Init logs storage
	if cfg.LogsStorage == "minio" {
		if cfg.LogsBucket == "" {
			log.DefaultLogger.Error("LOGS_BUCKET env var is not set")
		} else if ok, err := storageClient.IsConnectionPossible(ctx); ok && (err == nil) {
			log.DefaultLogger.Info("setting minio as logs storage")
			outputRepository = minioresult.NewMinioOutputRepository(storageClient, cfg.LogsBucket)
		} else {
			log.DefaultLogger.Infow("minio is not available, using default logs storage", "error", err)
		}
	}

	factory, err := repository.NewFactoryBuilder().WithMongoDB(repository.MongoDBFactoryConfig{
		Database:         db,
		AllowDiskUse:     cfg.APIMongoAllowDiskUse,
		IsDocDb:          cfg.APIMongoDBType == storage.TypeDocDB,
		LogGrpcClient:    logGrpcClient,
		OutputRepository: outputRepository,
	}).Build()
	if err != nil {
		return nil, err
	}

	return factory, nil
}

func CreatePostgresFactory(db *pgxpool.Pool) (repository.RepositoryFactory, error) {
	factory, err := repository.NewFactoryBuilder().WithPostgreSQL(repository.PostgreSQLFactoryConfig{
		Database: db,
	}).Build()
	if err != nil {
		return nil, err
	}

	return factory, nil
}
