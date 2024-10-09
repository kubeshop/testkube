package testworkflowexecutor

import (
	"bufio"
	"bytes"
	"context"
	errors2 "errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/log"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
	SaveResultRetryMaxAttempts = 100
	SaveResultRetryBaseDelay   = 300 * time.Millisecond

	SaveLogsRetryMaxAttempts = 10
	SaveLogsRetryBaseDelay   = 300 * time.Millisecond
)

//go:generate mockgen -destination=./mock_executor.go -package=testworkflowexecutor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor" TestWorkflowExecutor
type TestWorkflowExecutor interface {
	Control(ctx context.Context, testWorkflow *testworkflowsv1.TestWorkflow, execution *testkube.TestWorkflowExecution) error
	Recover(ctx context.Context)
	Execute(ctx context.Context, workflow testworkflowsv1.TestWorkflow, request testkube.TestWorkflowExecutionRequest) (
		execution testkube.TestWorkflowExecution, err error)
}

type executor struct {
	emitter                        *event.Emitter
	clientSet                      kubernetes.Interface
	repository                     testworkflow.Repository
	output                         testworkflow.OutputRepository
	testWorkflowTemplatesClient    testworkflowsclientv1.TestWorkflowTemplatesInterface
	processor                      testworkflowprocessor.Processor
	configMap                      configRepo.Repository
	testWorkflowExecutionsClient   testworkflowsclientv1.TestWorkflowExecutionsInterface
	testWorkflowsClient            testworkflowsclientv1.Interface
	metrics                        v1.Metrics
	secretManager                  secretmanager.SecretManager
	globalTemplateName             string
	apiUrl                         string
	namespace                      string
	defaultRegistry                string
	enableImageDataPersistentCache bool
	imageDataPersistentCacheKey    string
	dashboardURI                   string
	clusterID                      string
	serviceAccountNames            map[string]string
}

func New(emitter *event.Emitter,
	clientSet kubernetes.Interface,
	repository testworkflow.Repository,
	output testworkflow.OutputRepository,
	testWorkflowTemplatesClient testworkflowsclientv1.TestWorkflowTemplatesInterface,
	processor testworkflowprocessor.Processor,
	configMap configRepo.Repository,
	testWorkflowExecutionsClient testworkflowsclientv1.TestWorkflowExecutionsInterface,
	testWorkflowsClient testworkflowsclientv1.Interface,
	metrics v1.Metrics,
	secretManager secretmanager.SecretManager,
	serviceAccountNames map[string]string,
	globalTemplateName, namespace, apiUrl, defaultRegistry string,
	enableImageDataPersistentCache bool, imageDataPersistentCacheKey, dashboardURI, clusterID string) TestWorkflowExecutor {
	if serviceAccountNames == nil {
		serviceAccountNames = make(map[string]string)
	}

	return &executor{
		emitter:                        emitter,
		clientSet:                      clientSet,
		repository:                     repository,
		output:                         output,
		testWorkflowTemplatesClient:    testWorkflowTemplatesClient,
		processor:                      processor,
		configMap:                      configMap,
		testWorkflowExecutionsClient:   testWorkflowExecutionsClient,
		testWorkflowsClient:            testWorkflowsClient,
		metrics:                        metrics,
		secretManager:                  secretManager,
		serviceAccountNames:            serviceAccountNames,
		globalTemplateName:             globalTemplateName,
		apiUrl:                         apiUrl,
		namespace:                      namespace,
		defaultRegistry:                defaultRegistry,
		enableImageDataPersistentCache: enableImageDataPersistentCache,
		imageDataPersistentCacheKey:    imageDataPersistentCacheKey,
		dashboardURI:                   dashboardURI,
		clusterID:                      clusterID,
	}
}

func (e *executor) Deploy(ctx context.Context, bundle *testworkflowprocessor.Bundle) (err error) {
	return bundle.Deploy(ctx, e.clientSet, e.namespace)
}

func (e *executor) handleFatalError(execution *testkube.TestWorkflowExecution, err error, ts time.Time) {
	// Detect error type
	isAborted := errors.Is(err, testworkflowcontroller.ErrJobAborted)

	// Apply the expected result
	execution.Result.Fatal(err, isAborted, ts)
	err = e.repository.UpdateResult(context.Background(), execution.Id, execution.Result)
	if err != nil {
		log.DefaultLogger.Errorf("failed to save fatal error for execution %s: %v", execution.Id, err)
	}
	e.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
	go testworkflowcontroller.Cleanup(context.Background(), e.clientSet, execution.GetNamespace(e.namespace), execution.Id)
}

func (e *executor) Recover(ctx context.Context) {
	list, err := e.repository.GetRunning(ctx)
	if err != nil {
		return
	}
	for i := range list {
		go func(execution *testkube.TestWorkflowExecution) {
			var testWorkflow *testworkflowsv1.TestWorkflow
			var err error
			if execution.Workflow != nil {
				testWorkflow, err = e.testWorkflowsClient.Get(execution.Workflow.Name)
				if err != nil {
					e.handleFatalError(execution, err, time.Time{})
					return
				}
			}

			err = e.Control(context.Background(), testWorkflow, execution)
			if err != nil {
				e.handleFatalError(execution, err, time.Time{})
			}
		}(&list[i])
	}
}

func (e *executor) updateStatus(testWorkflow *testworkflowsv1.TestWorkflow, execution *testkube.TestWorkflowExecution,
	testWorkflowExecution *testworkflowsv1.TestWorkflowExecution) {
	if testWorkflow != nil {
		testWorkflow.Status = testworkflowmappers.MapTestWorkflowExecutionAPIToKubeTestWorkflowStatusSummary(execution)
		if err := e.testWorkflowsClient.UpdateStatus(testWorkflow); err != nil {
			log.DefaultLogger.Errorw("failed to update test workflow status", "error", err)
		}
	}

	if testWorkflowExecution != nil {
		testWorkflowExecution.Status = testworkflowmappers.MapTestWorkflowExecutionStatusAPIToKube(execution, testWorkflowExecution.Generation)
		if err := e.testWorkflowExecutionsClient.UpdateStatus(testWorkflowExecution); err != nil {
			log.DefaultLogger.Errorw("failed to update test workflow execution", "error", err)
		}
	}
}

func (e *executor) Control(ctx context.Context, testWorkflow *testworkflowsv1.TestWorkflow, execution *testkube.TestWorkflowExecution) error {
	ctrl, err := testworkflowcontroller.New(ctx, e.clientSet, execution.GetNamespace(e.namespace), execution.Id, execution.ScheduledAt)
	if err != nil {
		log.DefaultLogger.Errorw("failed to control the TestWorkflow", "id", execution.Id, "error", err)
		return err
	}
	defer ctrl.StopController()

	// Prepare stream for writing log
	r, writer := io.Pipe()
	reader := bufio.NewReader(r)
	ref := ""

	var testWorkflowExecution *testworkflowsv1.TestWorkflowExecution
	if execution.TestWorkflowExecutionName != "" {
		testWorkflowExecution, err = e.testWorkflowExecutionsClient.Get(execution.TestWorkflowExecutionName)
		if err != nil {
			log.DefaultLogger.Errorw("failed to get test workflow execution", "error", err)
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for v := range ctrl.Watch(ctx) {
			if v.Error != nil {
				log.DefaultLogger.Errorw("error from TestWorkflow watcher", "id", execution.Id, "error", v.Error)
				continue
			}
			if v.Value.Output != nil {
				if !v.Value.Temporary {
					execution.Output = append(execution.Output, *testworkflowcontroller.InstructionToInternal(v.Value.Output))
				}
			} else if v.Value.Result != nil {
				execution.Result = v.Value.Result
				if execution.Result.IsFinished() {
					execution.StatusAt = execution.Result.FinishedAt
				}
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					e.updateStatus(testWorkflow, execution, testWorkflowExecution)
					wg.Done()
				}()
				go func() {
					err := e.repository.UpdateResult(ctx, execution.Id, execution.Result)
					if err != nil {
						log.DefaultLogger.Error(errors.Wrap(err, "error saving test workflow execution result"))
					}
					wg.Done()
				}()
				wg.Wait()
			} else if !v.Value.Temporary {
				if ref != v.Value.Ref && v.Value.Ref != "" {
					ref = v.Value.Ref
					_, err := writer.Write([]byte(instructions.SprintHint(ref, initconstants.InstructionStart)))
					if err != nil {
						log.DefaultLogger.Error(errors.Wrap(err, "saving log output signature"))
					}
				}
				_, err := writer.Write([]byte(v.Value.Log))
				if err != nil {
					log.DefaultLogger.Error(errors.Wrap(err, "saving log output content"))
				}
			}
		}

		// Try to gracefully handle abort
		if execution.Result.FinishedAt.IsZero() {
			// Handle container failure
			abortedAt := time.Time{}
			for _, v := range execution.Result.Steps {
				if v.Status != nil && *v.Status == testkube.ABORTED_TestWorkflowStepStatus {
					abortedAt = v.FinishedAt
					break
				}
			}
			if !abortedAt.IsZero() {
				e.handleFatalError(execution, testworkflowcontroller.ErrJobAborted, abortedAt)
			} else {
				// Handle unknown state
				ctrl.StopController()
				ctrl, err = testworkflowcontroller.New(ctx, e.clientSet, execution.GetNamespace(e.namespace), execution.Id, execution.ScheduledAt)
				if err == nil {
					for v := range ctrl.Watch(ctx) {
						if v.Error != nil || v.Value.Output == nil {
							continue
						}

						execution.Result = v.Value.Result
						if execution.Result.IsFinished() {
							execution.StatusAt = execution.Result.FinishedAt
						}
						err := e.repository.UpdateResult(ctx, execution.Id, execution.Result)
						if err != nil {
							log.DefaultLogger.Error(errors.Wrap(err, "error saving test workflow execution result"))
						}
					}
				} else {
					e.handleFatalError(execution, err, time.Time{})
				}
			}
		}

		err := writer.Close()
		if err != nil {
			log.DefaultLogger.Errorw("failed to close TestWorkflow log output stream", "id", execution.Id, "error", err)
		}

		// TODO: Consider AppendOutput ($push) instead
		_ = e.repository.UpdateOutput(ctx, execution.Id, execution.Output)
		if execution.Result.IsFinished() {
			e.sendRunWorkflowTelemetry(ctx, testWorkflow, execution)

			if execution.Result.IsPassed() {
				e.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution))
			} else if execution.Result.IsAborted() {
				e.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
			} else {
				e.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
			}
		}
	}()

	// Stream the log into Minio
	err = e.output.SaveLog(context.Background(), execution.Id, execution.Workflow.Name, reader)

	// Retry saving the logs to Minio if something goes wrong
	for attempt := 1; err != nil && attempt <= SaveLogsRetryMaxAttempts; attempt++ {
		log.DefaultLogger.Errorw("retrying save of TestWorkflow log output", "id", execution.Id, "error", err)
		time.Sleep(SaveLogsRetryBaseDelay * time.Duration(attempt))
		err = e.output.SaveLog(context.Background(), execution.Id, execution.Workflow.Name, ctrl.Logs(context.Background(), false))
	}
	if err != nil {
		log.DefaultLogger.Errorw("failed to save TestWorkflow log output", "id", execution.Id, "error", err)
	}

	wg.Wait()

	e.metrics.IncAndObserveExecuteTestWorkflow(*execution, e.dashboardURI)

	e.updateStatus(testWorkflow, execution, testWorkflowExecution) // TODO: Consider if it is needed
	err = testworkflowcontroller.Cleanup(ctx, e.clientSet, execution.GetNamespace(e.namespace), execution.Id)
	if err != nil {
		log.DefaultLogger.Errorw("failed to cleanup TestWorkflow resources", "id", execution.Id, "error", err)
	}

	return nil
}

func (e *executor) getPreExecutionMachine(workflow *testworkflowsv1.TestWorkflow, orgId, envId string) expressions.Machine {
	controlPlaneConfig := e.buildControlPlaneConfig(orgId, envId)
	workflowConfig := e.buildWorkflowConfig(workflow)
	cloudMachine := testworkflowconfig.CreateCloudMachine(&controlPlaneConfig)
	workflowMachine := testworkflowconfig.CreateWorkflowMachine(&workflowConfig)
	return expressions.CombinedMachines(cloudMachine, workflowMachine)
}

func (e *executor) getPostExecutionMachine(execution *testkube.TestWorkflowExecution, orgId, envId, resourceId, rootResourceId, fsPrefix string) expressions.Machine {
	executionConfig := e.buildExecutionConfig(execution, orgId, envId)
	resourceConfig := e.buildResourceConfig(resourceId, rootResourceId, fsPrefix)
	resourceMachine := testworkflowconfig.CreateResourceMachine(&resourceConfig)
	executionMachine := testworkflowconfig.CreateExecutionMachine(&executionConfig)
	return expressions.CombinedMachines(executionMachine, resourceMachine)
}

func (e *executor) getWorkerMachine(namespace string) expressions.Machine {
	runtimeConfig := e.buildWorkerConfig(namespace)
	return testworkflowconfig.CreateWorkerMachine(&runtimeConfig)
}

func (e *executor) buildExecutionConfig(execution *testkube.TestWorkflowExecution, orgId, envId string) testworkflowconfig.ExecutionConfig {
	return testworkflowconfig.ExecutionConfig{
		Id:              execution.Id,
		Name:            execution.Name,
		Number:          execution.Number,
		ScheduledAt:     execution.ScheduledAt,
		DisableWebhooks: execution.DisableWebhooks,
		Tags:            execution.Tags,
		Debug:           false,
		OrganizationId:  orgId,
		EnvironmentId:   envId,
	}
}

func (e *executor) buildWorkflowConfig(workflow *testworkflowsv1.TestWorkflow) testworkflowconfig.WorkflowConfig {
	return testworkflowconfig.WorkflowConfig{
		Name:   workflow.Name,
		Labels: workflow.Labels,
	}
}

func (e *executor) buildResourceConfig(resourceId, rootResourceId, fsPrefix string) testworkflowconfig.ResourceConfig {
	return testworkflowconfig.ResourceConfig{
		Id:       resourceId,
		RootId:   rootResourceId,
		FsPrefix: fsPrefix,
	}
}

func (e *executor) buildWorkerConfig(namespace string) testworkflowconfig.WorkerConfig {
	duration, err := time.ParseDuration(common.GetOr(os.Getenv("TESTKUBE_IMAGE_CREDENTIALS_CACHE_TTL"), "30m"))
	if err != nil {
		duration = 30 * time.Minute
	}

	cloudUrl := common.GetOr(os.Getenv("TESTKUBE_PRO_URL"), os.Getenv("TESTKUBE_CLOUD_URL"))
	cloudApiKey := common.GetOr(os.Getenv("TESTKUBE_PRO_API_KEY"), os.Getenv("TESTKUBE_CLOUD_API_KEY"))
	if cloudApiKey == "" {
		cloudUrl = ""
	}

	return testworkflowconfig.WorkerConfig{
		Namespace:                         namespace,
		DefaultRegistry:                   e.defaultRegistry,
		DefaultServiceAccount:             e.serviceAccountNames[namespace],
		ClusterID:                         e.clusterID,
		InitImage:                         constants.DefaultInitImage,
		ToolkitImage:                      constants.DefaultToolkitImage,
		ImageInspectorPersistenceEnabled:  e.enableImageDataPersistentCache,
		ImageInspectorPersistenceCacheKey: e.imageDataPersistentCacheKey,
		ImageInspectorPersistenceCacheTTL: duration,

		Connection: testworkflowconfig.WorkerConnectionConfig{
			Url:         cloudUrl,
			ApiKey:      cloudApiKey,
			SkipVerify:  common.GetOr(os.Getenv("TESTKUBE_PRO_SKIP_VERIFY"), os.Getenv("TESTKUBE_CLOUD_SKIP_VERIFY"), "false") == "true",
			TlsInsecure: common.GetOr(os.Getenv("TESTKUBE_PRO_TLS_INSECURE"), os.Getenv("TESTKUBE_CLOUD_TLS_INSECURE"), "false") == "true",

			// TODO: Avoid
			LocalApiUrl: e.apiUrl,
			ObjectStorage: testworkflowconfig.ObjectStorageConfig{
				Endpoint:        os.Getenv("STORAGE_ENDPOINT"),
				AccessKeyID:     os.Getenv("STORAGE_ACCESSKEYID"),
				SecretAccessKey: os.Getenv("STORAGE_SECRETACCESSKEY"),
				Region:          os.Getenv("STORAGE_REGION"),
				Token:           os.Getenv("STORAGE_TOKEN"),
				Bucket:          os.Getenv("STORAGE_BUCKET"),
				Ssl:             common.GetOr(os.Getenv("STORAGE_SSL"), "false") == "true",
				SkipVerify:      common.GetOr(os.Getenv("STORAGE_SKIP_VERIFY"), "false") == "true",
				CertFile:        os.Getenv("STORAGE_CERT_FILE"),
				KeyFile:         os.Getenv("STORAGE_KEY_FILE"),
				CAFile:          os.Getenv("STORAGE_CA_FILE"),
			},
		},
	}
}

func (e *executor) buildControlPlaneConfig(orgId, envId string) testworkflowconfig.ControlPlaneConfig {
	dashboardUrl := e.dashboardURI
	if orgId != "" && envId != "" && dashboardUrl == "" {
		cloudUiUrl := common.GetOr(os.Getenv("TESTKUBE_PRO_UI_URL"), os.Getenv("TESTKUBE_CLOUD_UI_URL"))
		dashboardUrl = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard", cloudUiUrl, orgId, envId)
	}
	return testworkflowconfig.ControlPlaneConfig{
		DashboardUrl:   dashboardUrl,
		CDEventsTarget: os.Getenv("CDEVENTS_TARGET"),
	}
}

func (e *executor) initialize(ctx context.Context, workflow *testworkflowsv1.TestWorkflow, request *testkube.TestWorkflowExecutionRequest) (execution *testkube.TestWorkflowExecution, namespace string, secrets []corev1.Secret, err error) {
	// Delete unnecessary data
	delete(workflow.Annotations, "kubectl.kubernetes.io/last-applied-configuration")

	// Build the initial execution entity
	now := time.Now().UTC()
	executionId := primitive.NewObjectIDFromTimestamp(now).Hex()

	// Load execution identifier data
	// TODO: if request.Name is provided, consider checking for uniqueness early, and not incrementing the execution number.
	number, err := e.repository.GetNextExecutionNumber(context.Background(), workflow.Name)
	if err != nil {
		log.DefaultLogger.Errorw("failed to retrieve TestWorkflow execution number", "id", executionId, "error", err)
	}
	executionName := request.Name
	if executionName == "" {
		executionName = fmt.Sprintf("%s-%d", workflow.Name, number)
	}

	// Ensure the execution name is unique
	// TODO: Consider if we shouldn't make name unique across all TestWorkflows
	next, _ := e.repository.GetByNameAndTestWorkflow(ctx, executionName, workflow.Name)
	if next.Name == executionName {
		return execution, "", nil, errors.Wrap(err, "execution name already exists")
	}

	// Initialize the storage for dynamically created secrets
	secretsBatch := e.secretManager.Batch("twe-", executionId).ForceEnable()

	// Preserve initial workflow
	initialWorkflow := workflow.DeepCopy()
	initialWorkflowApi := testworkflowmappers.MapKubeToAPI(initialWorkflow)

	// Simplify the workflow data initially
	_ = expressions.Simplify(&workflow)

	// Create the execution entity
	execution = &testkube.TestWorkflowExecution{
		Id:          executionId,
		Name:        executionName,
		Number:      number,
		ScheduledAt: now,
		StatusAt:    now,
		Signature:   []testkube.TestWorkflowSignature{},
		Result: &testkube.TestWorkflowResult{
			Status:          common.Ptr(testkube.QUEUED_TestWorkflowStatus),
			PredictedStatus: common.Ptr(testkube.PASSED_TestWorkflowStatus),
			Initialization: &testkube.TestWorkflowStepResult{
				Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
			},
			Steps: map[string]testkube.TestWorkflowStepResult{},
		},
		Output:                    []testkube.TestWorkflowOutput{},
		Workflow:                  initialWorkflowApi,
		ResolvedWorkflow:          initialWorkflowApi,
		TestWorkflowExecutionName: request.TestWorkflowExecutionName,
		DisableWebhooks:           request.DisableWebhooks,
		Tags:                      map[string]string{},
	}

	// Try to resolve tags initially
	if workflow.Spec.Execution != nil {
		execution.Tags = workflow.Spec.Execution.Tags
	}
	execution.Tags = testworkflowresolver.MergeTags(execution.Tags, request.Tags)

	// Inject the global template
	if e.globalTemplateName != "" {
		testworkflowresolver.AddGlobalTemplateRef(workflow, testworkflowsv1.TemplateRef{
			Name: testworkflowresolver.GetDisplayTemplateName(e.globalTemplateName),
		})
	}

	// Apply the configuration
	_, err = testworkflowresolver.ApplyWorkflowConfig(workflow, testworkflowmappers.MapConfigValueAPIToKube(request.Config), secretsBatch.Append)
	if err != nil {
		execution.InitializationError("Failed to apply configuration.", err)
		return execution, "", nil, err
	}

	// Fetch all required templates
	tpls := testworkflowresolver.ListTemplates(workflow)
	tplsMap := make(map[string]testworkflowsv1.TestWorkflowTemplate, len(tpls))
	for tplName := range tpls {
		tpl, err := e.testWorkflowTemplatesClient.Get(tplName)
		if err != nil {
			execution.InitializationError(fmt.Sprintf("Failed to fetch '%s' template.", testworkflowresolver.GetDisplayTemplateName(tplName)), err)
			return execution, "", nil, err
		}
		tplsMap[tplName] = *tpl
	}

	// Resolve the TestWorkflow
	err = testworkflowresolver.ApplyTemplates(workflow, tplsMap, secretsBatch.Append)
	if err != nil {
		execution.InitializationError("Failed to apply templates.", err)
		return execution, "", nil, err
	}

	// Preserve resolved TestWorkflow
	resolvedWorkflow := workflow.DeepCopy()

	// Determine execution namespace
	// TODO: Should not default namespace be on runner?
	namespace = e.namespace
	if workflow.Spec.Job != nil && workflow.Spec.Job.Namespace != "" {
		namespace = workflow.Spec.Job.Namespace
	}
	if _, ok := e.serviceAccountNames[namespace]; !ok {
		execution.InitializationError(fmt.Sprintf("Not supported '%s' execution namespace.", namespace), err)
		return execution, "", nil, err
	}

	// Try to resolve the tags further
	if workflow.Spec.Execution != nil {
		execution.Tags = workflow.Spec.Execution.Tags
	}
	execution.Tags = testworkflowresolver.MergeTags(execution.Tags, request.Tags)

	// Apply more resolved data to the execution
	execution.Namespace = namespace // TODO: DELETE?
	execution.ResolvedWorkflow = testworkflowmappers.MapKubeToAPI(resolvedWorkflow)

	// Determine the organization/environment
	cloudApiKey := common.GetOr(os.Getenv("TESTKUBE_PRO_API_KEY"), os.Getenv("TESTKUBE_CLOUD_API_KEY"))
	environmentId := common.GetOr(os.Getenv("TESTKUBE_PRO_ENV_ID"), os.Getenv("TESTKUBE_CLOUD_ENV_ID"))
	organizationId := common.GetOr(os.Getenv("TESTKUBE_PRO_ORG_ID"), os.Getenv("TESTKUBE_CLOUD_ORG_ID"))
	if cloudApiKey == "" {
		organizationId = ""
		environmentId = ""
	}

	// Simplify the result
	preMachine := e.getPreExecutionMachine(workflow, organizationId, environmentId)
	postMachine := e.getPostExecutionMachine(execution, organizationId, environmentId, executionId, executionId, "")
	_ = expressions.Simplify(&workflow, preMachine, postMachine)

	// Build the final tags
	if workflow.Spec.Execution != nil {
		execution.Tags = workflow.Spec.Execution.Tags
	}
	execution.Tags = testworkflowresolver.MergeTags(execution.Tags, request.Tags)

	return execution, namespace, secretsBatch.Get(), nil
}

func (e *executor) notifyResult(execution *testkube.TestWorkflowExecution) {
	if !execution.Result.IsFinished() {
		return
	}
	if execution.Result.IsPassed() {
		e.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution))
	} else if execution.Result.IsAborted() {
		e.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
	} else {
		e.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
	}
}

func (e *executor) saveEmptyLogs(execution *testkube.TestWorkflowExecution) (err error) {
	if !execution.Result.IsFinished() {
		return nil
	}
	for i := 1; i <= SaveLogsRetryMaxAttempts; i++ {
		err = e.output.SaveLog(context.Background(), execution.Id, execution.Workflow.Name, bytes.NewReader(nil))
		if err == nil {
			return nil
		}
		log.DefaultLogger.Warnw("failed to save empty logs. retrying...", "id", execution.Id, "error", err)
		time.Sleep(time.Duration(i) * SaveResultRetryBaseDelay)
	}
	log.DefaultLogger.Errorw("failed to save empty logs", "id", execution.Id, "error", err)
	return err
}

func (e *executor) updateInDatabase(ctx context.Context, execution *testkube.TestWorkflowExecution) (err error) {
	for i := 1; i <= SaveResultRetryMaxAttempts; i++ {
		err = e.repository.Update(ctx, *execution)
		if err == nil {
			return nil
		}
		log.DefaultLogger.Warnw("failed to update execution. retrying...", "id", execution.Id, "error", err)
		time.Sleep(time.Duration(i) * SaveResultRetryBaseDelay)
	}
	log.DefaultLogger.Errorw("failed to update execution", "id", execution.Id, "error", err)
	return errors.Wrap(err, fmt.Sprintf("updating execution in storage: %s", err.Error()))
}

func (e *executor) updateInKubernetes(_ context.Context, execution *testkube.TestWorkflowExecution) (err error) {
	if execution.TestWorkflowExecutionName == "" {
		return nil
	}
	for i := 1; i <= SaveResultRetryMaxAttempts; i++ {
		// Load current object
		var cr *testworkflowsv1.TestWorkflowExecution
		cr, err = e.testWorkflowExecutionsClient.Get(execution.TestWorkflowExecutionName)
		if err == nil {
			cr.Status = testworkflowmappers.MapTestWorkflowExecutionStatusAPIToKube(execution, cr.Generation)
			if err := e.testWorkflowExecutionsClient.UpdateStatus(cr); err == nil {
				return nil
			}
		}
		log.DefaultLogger.Warnw("failed to update execution object in cluster. retrying...", "id", execution.Id, "error", err)
		time.Sleep(time.Duration(i) * SaveResultRetryBaseDelay)
	}
	log.DefaultLogger.Errorw("failed to update execution object in cluster", "id", execution.Id, "error", err)
	return errors.Wrap(err, fmt.Sprintf("updating execution object in cluster: %s", err.Error()))
}

func (e *executor) update(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
	var wg sync.WaitGroup
	wg.Add(2)

	// TODO: Update also TestWorkflow.Status in Kubernetes
	var err1, err2 error
	go func() {
		err1 = e.updateInDatabase(ctx, execution)
		wg.Done()
	}()
	go func() {
		err2 = e.updateInKubernetes(ctx, execution)
		wg.Done()
	}()
	wg.Wait()

	return errors2.Join(err1, err2)
}

func (e *executor) insert(ctx context.Context, execution *testkube.TestWorkflowExecution) (err error) {
	for i := 1; i <= SaveResultRetryMaxAttempts; i++ {
		err = e.repository.Insert(ctx, *execution)
		if err == nil {
			return nil
		}
		log.DefaultLogger.Warnw("failed to insert execution. retrying...", "id", execution.Id, "error", err)
		time.Sleep(time.Duration(i) * SaveResultRetryBaseDelay)
	}
	log.DefaultLogger.Errorw("failed to insert execution", "id", execution.Id, "error", err)
	return errors.Wrap(err, fmt.Sprintf("inserting execution in storage: %s", err.Error()))
}

func (e *executor) Execute(ctx context.Context, workflow testworkflowsv1.TestWorkflow, request testkube.TestWorkflowExecutionRequest) (
	testkube.TestWorkflowExecution, error) {
	execution, namespace, secrets, err := e.initialize(ctx, &workflow, &request)

	// Handle error without execution built
	if execution == nil {
		return testkube.TestWorkflowExecution{}, err
	}

	// Insert the execution
	insertErr := e.insert(context.Background(), execution)
	if insertErr != nil {
		e.saveEmptyLogs(execution)
		if err != nil {
			return *execution, errors.Wrap(insertErr, fmt.Sprintf("initializing error: %s: saving", err.Error()))
		}
		return *execution, insertErr
	}
	e.emitter.Notify(testkube.NewEventQueueTestWorkflow(execution))

	// TODO: Check if we need to resolve the [control plane] secrets (?)

	// Send events
	defer e.notifyResult(execution)

	// Handle finished execution (i.e. initialization error)
	if execution.Result.IsFinished() {
		e.saveEmptyLogs(execution)
		e.updateInKubernetes(ctx, execution)
		return *execution, nil
	}

	// Determine the organization/environment
	cloudApiKey := common.GetOr(os.Getenv("TESTKUBE_PRO_API_KEY"), os.Getenv("TESTKUBE_CLOUD_API_KEY"))
	environmentId := common.GetOr(os.Getenv("TESTKUBE_PRO_ENV_ID"), os.Getenv("TESTKUBE_CLOUD_ENV_ID"))
	organizationId := common.GetOr(os.Getenv("TESTKUBE_PRO_ORG_ID"), os.Getenv("TESTKUBE_CLOUD_ORG_ID"))
	if cloudApiKey == "" {
		organizationId = ""
		environmentId = ""
	}

	//// Simplify the workflow
	//preMachine := e.getPreExecutionMachine(&workflow, organizationId, environmentId)
	//postMachine := e.getPostExecutionMachine(execution, organizationId, environmentId, execution.Id, execution.Id, "")
	//runtimeMachine := e.getWorkerMachine(namespace)

	// Apply default service account
	if workflow.Spec.Pod == nil {
		workflow.Spec.Pod = &testworkflowsv1.PodConfig{}
	}
	if workflow.Spec.Pod.ServiceAccountName == "" {
		workflow.Spec.Pod.ServiceAccountName = "{{internal.serviceaccount.default}}"
	}

	// Build the toolkit configuration
	internalConfig := testworkflowconfig.InternalConfig{
		Execution:    e.buildExecutionConfig(execution, organizationId, environmentId),
		Workflow:     e.buildWorkflowConfig(&workflow),
		Resource:     e.buildResourceConfig(execution.Id, execution.Id, ""),
		ControlPlane: e.buildControlPlaneConfig(organizationId, environmentId),
		Worker:       e.buildWorkerConfig(namespace),
	}

	// Process the TestWorkflow
	bundle, err := e.processor.Bundle(ctx, &workflow, testworkflowprocessor.BundleOptions{Config: internalConfig, Secrets: secrets})
	if err != nil {
		defer e.saveEmptyLogs(execution)
		execution.InitializationError("Failed to process Test Workflow.", err)
		return *execution, errors.Wrap(e.update(context.Background(), execution), fmt.Sprintf("processing error: %s: saving", err.Error()))
	}

	// Apply the signature
	execution.Signature = stage.MapSignatureListToInternal(bundle.Signature)
	execution.Result.Steps = stage.MapSignatureListToStepResults(bundle.Signature)
	// TODO: Consider applying to the TestWorkflowExecution object in Kubernetes
	updateErr := e.repository.Update(context.Background(), *execution)
	if updateErr != nil {
		e.saveEmptyLogs(execution)
		return *execution, e.update(context.Background(), execution)
	}

	// Inform about execution start TODO: Consider
	//e.emitter.Notify(testkube.NewEventStartTestWorkflow(execution))

	// Deploy required resources
	err = e.Deploy(context.Background(), bundle)
	if err != nil {
		defer e.saveEmptyLogs(execution)
		execution.InitializationError("Failed to deploy the execution resources.", err)
		return *execution, errors.Wrap(e.update(context.Background(), execution), fmt.Sprintf("deployment error: %s: saving", err.Error()))
	}

	// Start to control the results
	go func() {
		// TODO: Use OpenAPI objects only
		err = e.Control(context.Background(), testworkflowmappers.MapAPIToKube(execution.Workflow), execution)
		if err != nil {
			e.handleFatalError(execution, err, time.Time{})
			return
		}
	}()

	return *execution, nil
}
