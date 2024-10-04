package testworkflowexecutor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
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
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
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

func (e *executor) getMachine(workflow *testworkflowsv1.TestWorkflow, namespace string, resourceId, rootResourceId, fsPrefix string) expressions.Machine {
	// Prepare all the data for resolving Test Workflow
	storageMachine := createStorageMachine()
	cloudMachine := createCloudMachine()
	workflowMachine := createWorkflowMachine(workflow)
	resourceMachine := createResourceMachine(resourceId, rootResourceId, fsPrefix)
	restMachine := expressions.NewMachine().
		RegisterStringMap("internal", map[string]string{
			"serviceaccount.default": e.serviceAccountNames[namespace],

			"api.url":         e.apiUrl,
			"namespace":       namespace,
			"defaultRegistry": e.defaultRegistry,
			"clusterId":       e.clusterID,
			"cdeventsTarget":  os.Getenv("CDEVENTS_TARGET"),

			"images.defaultRegistry":     e.defaultRegistry,
			"images.init":                constants.DefaultInitImage,
			"images.toolkit":             constants.DefaultToolkitImage,
			"images.persistence.enabled": strconv.FormatBool(e.enableImageDataPersistentCache),
			"images.persistence.key":     e.imageDataPersistentCacheKey,
			"images.cache.ttl":           common.GetOr(os.Getenv("TESTKUBE_IMAGE_CREDENTIALS_CACHE_TTL"), "30m"),
		})
	return expressions.CombinedMachines(storageMachine, cloudMachine, workflowMachine, resourceMachine, restMachine)
}

func (e *executor) getExecutionMachine(execution *testkube.TestWorkflowExecution) expressions.Machine {
	// Build machine with actual execution data
	return expressions.NewMachine().Register("execution", map[string]interface{}{
		"id":              execution.Id,
		"name":            execution.Name,
		"number":          execution.Number,
		"scheduledAt":     execution.ScheduledAt.Format(constants.RFC3339Millis),
		"disableWebhooks": execution.DisableWebhooks,
		"tags":            execution.Tags,
	})
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

	// Apply default service account
	if workflow.Spec.Pod == nil {
		workflow.Spec.Pod = &testworkflowsv1.PodConfig{}
	}
	if workflow.Spec.Pod.ServiceAccountName == "" {
		workflow.Spec.Pod.ServiceAccountName = "{{internal.serviceaccount.default}}"
	}

	// Try to resolve the tags further
	if workflow.Spec.Execution != nil {
		execution.Tags = workflow.Spec.Execution.Tags
	}
	execution.Tags = testworkflowresolver.MergeTags(execution.Tags, request.Tags)

	// Apply more resolved data to the execution
	execution.Namespace = namespace // TODO: DELETE?
	execution.ResolvedWorkflow = testworkflowmappers.MapKubeToAPI(resolvedWorkflow)

	// Simplify the workflow
	machine := e.getMachine(workflow, namespace, executionId, executionId, "")
	executionMachine := e.getExecutionMachine(execution)
	_ = expressions.Simplify(&workflow, machine, executionMachine)

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

func (e *executor) saveEmptyLogs(execution *testkube.TestWorkflowExecution) {
	if !execution.Result.IsFinished() {
		return
	}
	err := e.output.SaveLog(context.Background(), execution.Id, execution.Workflow.Name, bytes.NewReader(nil))
	if err != nil {
		// TODO: Retry on error
		log.DefaultLogger.Errorw("failed to save empty logs", "id", execution.Id, "error", err)
	}
}

func (e *executor) Execute(ctx context.Context, workflow testworkflowsv1.TestWorkflow, request testkube.TestWorkflowExecutionRequest) (
	testkube.TestWorkflowExecution, error) {
	execution, namespace, secrets, err := e.initialize(ctx, &workflow, &request)

	// Handle error without execution built
	if execution == nil {
		return testkube.TestWorkflowExecution{}, err
	}

	// Insert the execution
	// TODO: Consider applying to the TestWorkflowExecution object in Kubernetes
	insertErr := e.repository.Insert(ctx, *execution)
	if insertErr != nil {
		e.saveEmptyLogs(execution)
		if err != nil {
			return *execution, errors.Wrap(insertErr, fmt.Sprintf("inserting execution to storage after error: %s", err.Error()))
		}
		// FIXME: Retry insert on error
		return *execution, errors.Wrap(insertErr, "inserting execution to storage")
	}
	e.emitter.Notify(testkube.NewEventQueueTestWorkflow(execution))

	// Send events
	defer e.notifyResult(execution)

	// Handle finished execution (i.e. initialization error)
	if execution.Result.IsFinished() {
		e.saveEmptyLogs(execution)
		return *execution, nil
	}

	// Process and deploy the execution
	machine := e.getMachine(&workflow, namespace, execution.Id, execution.Id, "")
	executionMachine := e.getExecutionMachine(execution)

	// Process the TestWorkflow
	bundle, err := e.processor.Bundle(ctx, &workflow, testworkflowprocessor.BundleOptions{Secrets: secrets}, machine, executionMachine)
	if err != nil {
		defer e.saveEmptyLogs(execution)
		execution.InitializationError("Failed to process Test Workflow.", err)
		// TODO: Consider applying to the TestWorkflowExecution object in Kubernetes
		updateErr := e.repository.UpdateResult(context.Background(), execution.Id, execution.Result)
		if updateErr != nil {
			// FIXME: Retry update on error
			return *execution, errors.Wrap(updateErr, fmt.Sprintf("updating execution in storage after error: %s", err.Error()))
		}
		return *execution, nil
	}

	// Apply the signature
	execution.Signature = stage.MapSignatureListToInternal(bundle.Signature)
	execution.Result.Steps = stage.MapSignatureListToStepResults(bundle.Signature)
	// TODO: Consider applying to the TestWorkflowExecution object in Kubernetes
	updateErr := e.repository.Update(context.Background(), *execution)
	if updateErr != nil {
		e.saveEmptyLogs(execution)
		// FIXME: Retry update on error
		return *execution, errors.Wrap(updateErr, "updating execution in storage")
	}

	// Inform about execution start TODO: Consider
	//e.emitter.Notify(testkube.NewEventStartTestWorkflow(execution))

	// Deploy required resources
	err = e.Deploy(context.Background(), bundle)
	if err != nil {
		defer e.saveEmptyLogs(execution)
		execution.InitializationError("Failed to deploy the execution resources.", err)
		// TODO: Consider applying to the TestWorkflowExecution object in Kubernetes
		updateErr := e.repository.UpdateResult(context.Background(), execution.Id, execution.Result)
		if updateErr != nil {
			// FIXME: Retry update on error
			return *execution, errors.Wrap(updateErr, fmt.Sprintf("updating execution result in storage after error: %s", err.Error()))
		}
		return *execution, nil
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
