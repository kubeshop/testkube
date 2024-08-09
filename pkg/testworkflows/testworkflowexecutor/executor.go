package testworkflowexecutor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/log"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
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
	globalTemplateName             string
	apiUrl                         string
	namespace                      string
	defaultRegistry                string
	enableImageDataPersistentCache bool
	imageDataPersistentCacheKey    string
	dashboardURI                   string
	clusterID                      string
	runnerID                       string
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
	serviceAccountNames map[string]string,
	globalTemplateName, namespace, apiUrl, defaultRegistry string,
	enableImageDataPersistentCache bool, imageDataPersistentCacheKey, dashboardURI, clusterID, runnerID string) TestWorkflowExecutor {
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
		serviceAccountNames:            serviceAccountNames,
		globalTemplateName:             globalTemplateName,
		apiUrl:                         apiUrl,
		namespace:                      namespace,
		defaultRegistry:                defaultRegistry,
		enableImageDataPersistentCache: enableImageDataPersistentCache,
		imageDataPersistentCacheKey:    imageDataPersistentCacheKey,
		dashboardURI:                   dashboardURI,
		clusterID:                      clusterID,
		runnerID:                       runnerID,
	}
}

func (e *executor) Deploy(ctx context.Context, bundle *testworkflowprocessor.Bundle) (err error) {
	return bundle.Deploy(ctx, e.clientSet, e.namespace)
}

func (e *executor) handleFatalError(execution *testkube.TestWorkflowExecution, err error, ts time.Time) {
	// Detect error type
	isAborted := errors.Is(err, testworkflowcontroller.ErrJobAborted)
	isTimeout := errors.Is(err, testworkflowcontroller.ErrJobTimeout)

	// Build error timestamp, adjusting it for aborting job
	if ts.IsZero() {
		ts = time.Now()
		if isAborted || isTimeout {
			ts = ts.Add(-1 * testworkflowcontroller.DefaultInitTimeout)
		}
	}

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

func (e *executor) Execute(ctx context.Context, workflow testworkflowsv1.TestWorkflow, request testkube.TestWorkflowExecutionRequest) (execution testkube.TestWorkflowExecution, err error) {
	// Delete unnecessary data
	delete(workflow.Annotations, "kubectl.kubernetes.io/last-applied-configuration")

	// Preserve initial workflow
	initialWorkflow := workflow.DeepCopy()

	// Fetch the templates
	tpls := testworkflowresolver.ListTemplates(&workflow)
	tplsMap := make(map[string]testworkflowsv1.TestWorkflowTemplate, len(tpls))
	for tplName := range tpls {
		tpl, err := e.testWorkflowTemplatesClient.Get(tplName)
		if err != nil {
			return execution, errors.Wrap(err, "fetching error")
		}
		tplsMap[tplName] = *tpl
	}

	// Fetch the global template
	globalTemplateRef := testworkflowsv1.TemplateRef{}
	if e.globalTemplateName != "" {
		internalName := testworkflowresolver.GetInternalTemplateName(e.globalTemplateName)
		displayName := testworkflowresolver.GetDisplayTemplateName(e.globalTemplateName)

		if _, ok := tplsMap[internalName]; !ok {
			globalTemplatePtr, err := e.testWorkflowTemplatesClient.Get(internalName)
			if err != nil && !k8serrors.IsNotFound(err) {
				return execution, errors.Wrap(err, "global template error")
			} else if err == nil {
				tplsMap[internalName] = *globalTemplatePtr
			}
		}
		if _, ok := tplsMap[internalName]; ok {
			globalTemplateRef = testworkflowsv1.TemplateRef{Name: displayName}
			workflow.Spec.Use = append([]testworkflowsv1.TemplateRef{globalTemplateRef}, workflow.Spec.Use...)
		}
	}

	// Apply the configuration
	_, err = testworkflowresolver.ApplyWorkflowConfig(&workflow, testworkflowmappers.MapConfigValueAPIToKube(request.Config))
	if err != nil {
		return execution, errors.Wrap(err, "configuration")
	}

	// Resolve the TestWorkflow
	err = testworkflowresolver.ApplyTemplates(&workflow, tplsMap)
	if err != nil {
		return execution, errors.Wrap(err, "resolving error")
	}

	// Apply global template to parallel steps
	if globalTemplateRef.Name != "" {
		testworkflowresolver.AddGlobalTemplateRef(&workflow, globalTemplateRef)
		workflow.Spec.Use = nil
		err = testworkflowresolver.ApplyTemplates(&workflow, tplsMap)
		if err != nil {
			return execution, errors.Wrap(err, "resolving with global templates error")
		}
	}

	namespace := e.namespace
	if workflow.Spec.Job != nil && workflow.Spec.Job.Namespace != "" {
		namespace = workflow.Spec.Job.Namespace
	}

	if _, ok := e.serviceAccountNames[namespace]; !ok {
		return execution, fmt.Errorf("not supported execution namespace %s", namespace)
	}

	// Determine the dashboard information
	cloudApiKey := common.GetOr(os.Getenv("TESTKUBE_PRO_API_KEY"), os.Getenv("TESTKUBE_CLOUD_API_KEY"))
	cloudOrgId := common.GetOr(os.Getenv("TESTKUBE_PRO_ORG_ID"), os.Getenv("TESTKUBE_CLOUD_ORG_ID"))
	cloudEnvId := common.GetOr(os.Getenv("TESTKUBE_PRO_ENV_ID"), os.Getenv("TESTKUBE_CLOUD_ENV_ID"))
	cloudUiUrl := common.GetOr(os.Getenv("TESTKUBE_PRO_UI_URL"), os.Getenv("TESTKUBE_CLOUD_UI_URL"))
	dashboardUrl := env.Config().System.DashboardUrl
	if env.Config().Cloud.ApiKey != "" {
		dashboardUrl = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard",
			cloudUiUrl, env.Config().Cloud.OrgId, env.Config().Cloud.EnvId)
	}

	// Build the basic Execution data
	id := primitive.NewObjectID().Hex()
	now := time.Now()
	machine := expressions.NewMachine().
		RegisterStringMap("internal", map[string]string{
			"storage.url":        os.Getenv("STORAGE_ENDPOINT"),
			"storage.accessKey":  os.Getenv("STORAGE_ACCESSKEYID"),
			"storage.secretKey":  os.Getenv("STORAGE_SECRETACCESSKEY"),
			"storage.region":     os.Getenv("STORAGE_REGION"),
			"storage.bucket":     os.Getenv("STORAGE_BUCKET"),
			"storage.token":      os.Getenv("STORAGE_TOKEN"),
			"storage.ssl":        common.GetOr(os.Getenv("STORAGE_SSL"), "false"),
			"storage.skipVerify": common.GetOr(os.Getenv("STORAGE_SKIP_VERIFY"), "false"),
			"storage.certFile":   os.Getenv("STORAGE_CERT_FILE"),
			"storage.keyFile":    os.Getenv("STORAGE_KEY_FILE"),
			"storage.caFile":     os.Getenv("STORAGE_CA_FILE"),

			"cloud.enabled":         strconv.FormatBool(os.Getenv("TESTKUBE_PRO_API_KEY") != "" || os.Getenv("TESTKUBE_CLOUD_API_KEY") != ""),
			"cloud.api.key":         cloudApiKey,
			"cloud.api.tlsInsecure": common.GetOr(os.Getenv("TESTKUBE_PRO_TLS_INSECURE"), os.Getenv("TESTKUBE_CLOUD_TLS_INSECURE"), "false"),
			"cloud.api.skipVerify":  common.GetOr(os.Getenv("TESTKUBE_PRO_SKIP_VERIFY"), os.Getenv("TESTKUBE_CLOUD_SKIP_VERIFY"), "false"),
			"cloud.api.url":         common.GetOr(os.Getenv("TESTKUBE_PRO_URL"), os.Getenv("TESTKUBE_CLOUD_URL")),
			"cloud.ui.url":          cloudUiUrl,
			"cloud.api.orgId":       cloudOrgId,
			"cloud.api.envId":       cloudEnvId,

			"serviceaccount.default": e.serviceAccountNames[namespace],

			"dashboard.url":   os.Getenv("TESTKUBE_DASHBOARD_URI"),
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
		}).
		Register("workflow", map[string]string{
			"name": workflow.Name,
		}).
		Register("resource", map[string]string{
			"id":       id,
			"root":     id,
			"fsPrefix": "",
		}).
		Register("dashboard", map[string]string{
			"url": dashboardUrl,
		}).
		Register("organization", map[string]string{
			"id": cloudOrgId,
		}).
		Register("environment", map[string]string{
			"id": cloudEnvId,
		})
	mockExecutionMachine := expressions.NewMachine().Register("execution", map[string]interface{}{
		"id":              id,
		"name":            "<mock_name>",
		"number":          "1",
		"scheduledAt":     now.UTC().Format(constants.RFC3339Millis),
		"disableWebhooks": request.DisableWebhooks,
	})

	// Preserve resolved TestWorkflow
	resolvedWorkflow := workflow.DeepCopy()

	// Apply default service account
	if workflow.Spec.Pod == nil {
		workflow.Spec.Pod = &testworkflowsv1.PodConfig{}
	}
	if workflow.Spec.Pod.ServiceAccountName == "" {
		workflow.Spec.Pod.ServiceAccountName = "{{internal.serviceaccount.default}}"
	}

	// Validate the TestWorkflow
	_, err = e.processor.Bundle(ctx, workflow.DeepCopy(), machine, mockExecutionMachine)
	if err != nil {
		return execution, errors.Wrap(err, "processing error")
	}

	// Load execution identifier data
	number, err := e.repository.GetNextExecutionNumber(context.Background(), workflow.Name)
	if err != nil {
		log.DefaultLogger.Errorw("failed to retrieve TestWorkflow execution number", "id", id, "error", err)
	}

	executionName := request.Name
	if executionName == "" {
		executionName = fmt.Sprintf("%s-%d", workflow.Name, number)
	}

	testWorkflowExecutionName := request.TestWorkflowExecutionName
	// Ensure it is unique name
	// TODO: Consider if we shouldn't make name unique across all TestWorkflows
	next, _ := e.repository.GetByNameAndTestWorkflow(ctx, executionName, workflow.Name)
	if next.Name == executionName {
		return execution, errors.Wrap(err, "execution name already exists")
	}

	// Build machine with actual execution data
	executionMachine := expressions.NewMachine().Register("execution", map[string]interface{}{
		"id":              id,
		"name":            executionName,
		"number":          number,
		"scheduledAt":     now.UTC().Format(constants.RFC3339Millis),
		"disableWebhooks": request.DisableWebhooks,
	})

	// Process the TestWorkflow
	bundle, err := e.processor.Bundle(ctx, &workflow, machine, executionMachine)
	if err != nil {
		return execution, errors.Wrap(err, "processing error")
	}

	// Build Execution entity
	// TODO: Consider storing "config" as well
	execution = testkube.TestWorkflowExecution{
		Id:          id,
		Name:        executionName,
		Namespace:   namespace,
		Number:      number,
		ScheduledAt: now,
		StatusAt:    now,
		Signature:   stage.MapSignatureListToInternal(bundle.Signature),
		Result: &testkube.TestWorkflowResult{
			Status:          common.Ptr(testkube.QUEUED_TestWorkflowStatus),
			PredictedStatus: common.Ptr(testkube.PASSED_TestWorkflowStatus),
			Initialization: &testkube.TestWorkflowStepResult{
				Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
			},
			Steps: stage.MapSignatureListToStepResults(bundle.Signature),
		},
		Output:                    []testkube.TestWorkflowOutput{},
		Workflow:                  testworkflowmappers.MapKubeToAPI(initialWorkflow),
		ResolvedWorkflow:          testworkflowmappers.MapKubeToAPI(resolvedWorkflow),
		TestWorkflowExecutionName: testWorkflowExecutionName,
		DisableWebhooks:           request.DisableWebhooks,
		RunnerId:                  e.runnerID,
		RunningContext:            request.RunningContext,
	}

	log.DefaultLogger.Infow("inserting execution", "execution", execution, "runningContext", request.RunningContext)

	err = e.repository.Insert(ctx, execution)
	if err != nil {
		return execution, errors.Wrap(err, "inserting execution to storage")
	}

	// Inform about execution start
	e.emitter.Notify(testkube.NewEventQueueTestWorkflow(&execution))

	// Deploy required resources
	err = e.Deploy(context.Background(), bundle)
	if err != nil {
		e.handleFatalError(&execution, err, time.Time{})
		return execution, errors.Wrap(err, "deploying required resources")
	}

	// Start to control the results
	go func() {
		err = e.Control(context.Background(), initialWorkflow, &execution)
		if err != nil {
			e.handleFatalError(&execution, err, time.Time{})
			return
		}
	}()

	return execution, nil
}
