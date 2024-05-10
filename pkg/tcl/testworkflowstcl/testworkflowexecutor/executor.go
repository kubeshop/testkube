// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	initconstants "github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-init/data"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/log"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/tcl/expressionstcl"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowresolver"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/version"
)

//go:generate mockgen -destination=./mock_executor.go -package=testworkflowexecutor "github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowexecutor" TestWorkflowExecutor
type TestWorkflowExecutor interface {
	Control(ctx context.Context, execution *testkube.TestWorkflowExecution) error
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
	imageInspector                 imageinspector.Inspector
	configMap                      configRepo.Repository
	executionResults               result.Repository
	globalTemplateName             string
	apiUrl                         string
	namespace                      string
	defaultRegistry                string
	enableImageDataPersistentCache bool
	imageDataPersistentCacheKey    string
	serviceAccountNames            map[string]string
}

func New(emitter *event.Emitter,
	clientSet kubernetes.Interface,
	repository testworkflow.Repository,
	output testworkflow.OutputRepository,
	testWorkflowTemplatesClient testworkflowsclientv1.TestWorkflowTemplatesInterface,
	imageInspector imageinspector.Inspector,
	configMap configRepo.Repository,
	executionResults result.Repository,
	serviceAccountNames map[string]string,
	globalTemplateName, namespace, apiUrl, defaultRegistry string,
	enableImageDataPersistentCache bool, imageDataPersistentCacheKey string) TestWorkflowExecutor {
	if serviceAccountNames == nil {
		serviceAccountNames = make(map[string]string)
	}

	return &executor{
		emitter:                        emitter,
		clientSet:                      clientSet,
		repository:                     repository,
		output:                         output,
		testWorkflowTemplatesClient:    testWorkflowTemplatesClient,
		imageInspector:                 imageInspector,
		configMap:                      configMap,
		executionResults:               executionResults,
		serviceAccountNames:            serviceAccountNames,
		globalTemplateName:             globalTemplateName,
		apiUrl:                         apiUrl,
		namespace:                      namespace,
		defaultRegistry:                defaultRegistry,
		enableImageDataPersistentCache: enableImageDataPersistentCache,
		imageDataPersistentCacheKey:    imageDataPersistentCacheKey,
	}
}

func (e *executor) Deploy(ctx context.Context, bundle *testworkflowprocessor.Bundle) (err error) {
	namespace := e.namespace
	if bundle.Job.Namespace != "" {
		namespace = bundle.Job.Namespace
	}

	for _, item := range bundle.Secrets {
		_, err = e.clientSet.CoreV1().Secrets(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return
		}
	}
	for _, item := range bundle.ConfigMaps {
		_, err = e.clientSet.CoreV1().ConfigMaps(namespace).Create(ctx, &item, metav1.CreateOptions{})
		if err != nil {
			return
		}
	}
	_, err = e.clientSet.BatchV1().Jobs(namespace).Create(ctx, &bundle.Job, metav1.CreateOptions{})
	return
}

func (e *executor) handleFatalError(execution *testkube.TestWorkflowExecution, err error, ts time.Time) {
	// Detect error type
	isAborted := errors.Is(err, testworkflowcontroller.ErrJobAborted)
	isTimeout := errors.Is(err, testworkflowcontroller.ErrJobTimeout)

	// Build error timestamp, adjusting it for aborting job
	if ts.IsZero() {
		ts = time.Now()
		if isAborted || isTimeout {
			ts = ts.Truncate(testworkflowcontroller.DefaultInitTimeout)
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
			err := e.Control(context.Background(), execution)
			if err != nil {
				e.handleFatalError(execution, err, time.Time{})
			}
		}(&list[i])
	}
}

func (e *executor) Control(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
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
				execution.Output = append(execution.Output, *testworkflowcontroller.InstructionToInternal(v.Value.Output))
			} else if v.Value.Result != nil {
				execution.Result = v.Value.Result
				if execution.Result.IsFinished() {
					execution.StatusAt = execution.Result.FinishedAt
				}
				err := e.repository.UpdateResult(ctx, execution.Id, execution.Result)
				if err != nil {
					log.DefaultLogger.Error(errors.Wrap(err, "error saving test workflow execution result"))
				}
			} else {
				if ref != v.Value.Ref {
					ref = v.Value.Ref
					_, err := writer.Write([]byte(data.SprintHint(ref, initconstants.InstructionStart)))
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

	err = testworkflowcontroller.Cleanup(ctx, e.clientSet, execution.GetNamespace(e.namespace), execution.Id)
	if err != nil {
		log.DefaultLogger.Errorw("failed to cleanup TestWorkflow resources", "id", execution.Id, "error", err)
	}

	return nil
}

func (e *executor) Execute(ctx context.Context, workflow testworkflowsv1.TestWorkflow, request testkube.TestWorkflowExecutionRequest) (
	execution testkube.TestWorkflowExecution, err error) {
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

	// Build the basic Execution data
	id := primitive.NewObjectID().Hex()
	now := time.Now()
	machine := expressionstcl.NewMachine().
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
			"cloud.api.key":         common.GetOr(os.Getenv("TESTKUBE_PRO_API_KEY"), os.Getenv("TESTKUBE_CLOUD_API_KEY")),
			"cloud.api.tlsInsecure": common.GetOr(os.Getenv("TESTKUBE_PRO_TLS_INSECURE"), os.Getenv("TESTKUBE_CLOUD_TLS_INSECURE"), "false"),
			"cloud.api.skipVerify":  common.GetOr(os.Getenv("TESTKUBE_PRO_SKIP_VERIFY"), os.Getenv("TESTKUBE_CLOUD_SKIP_VERIFY"), "false"),
			"cloud.api.url":         common.GetOr(os.Getenv("TESTKUBE_PRO_URL"), os.Getenv("TESTKUBE_CLOUD_URL")),

			"dashboard.url":   os.Getenv("TESTKUBE_DASHBOARD_URI"),
			"api.url":         e.apiUrl,
			"namespace":       namespace,
			"defaultRegistry": e.defaultRegistry,

			"images.init":                constants.DefaultInitImage,
			"images.toolkit":             constants.DefaultToolkitImage,
			"images.persistence.enabled": strconv.FormatBool(e.enableImageDataPersistentCache),
			"images.persistence.key":     e.imageDataPersistentCacheKey,
		}).
		RegisterStringMap("workflow", map[string]string{
			"name": workflow.Name,
		}).
		RegisterStringMap("execution", map[string]string{
			"id": id,
		})

	// Preserve resolved TestWorkflow
	resolvedWorkflow := workflow.DeepCopy()

	// Process the TestWorkflow
	bundle, err := testworkflowprocessor.NewFullFeatured(e.imageInspector).
		Bundle(ctx, &workflow, machine)
	if err != nil {
		return execution, errors.Wrap(err, "processing error")
	}

	// Load execution identifier data
	// TODO: Consider if that should not be shared (as now it is between Tests and Test Suites)
	number, _ := e.executionResults.GetNextExecutionNumber(context.Background(), workflow.Name)
	executionName := request.Name
	if executionName == "" {
		executionName = fmt.Sprintf("%s-%d", workflow.Name, number)
	}

	// Ensure it is unique name
	// TODO: Consider if we shouldn't make name unique across all TestWorkflows
	next, _ := e.repository.GetByNameAndTestWorkflow(ctx, executionName, workflow.Name)
	if next.Name == executionName {
		return execution, errors.Wrap(err, "execution name already exists")
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
		Signature:   testworkflowprocessor.MapSignatureListToInternal(bundle.Signature),
		Result: &testkube.TestWorkflowResult{
			Status:          common.Ptr(testkube.QUEUED_TestWorkflowStatus),
			PredictedStatus: common.Ptr(testkube.PASSED_TestWorkflowStatus),
			Initialization: &testkube.TestWorkflowStepResult{
				Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
			},
			Steps: testworkflowprocessor.MapSignatureListToStepResults(bundle.Signature),
		},
		Output:           []testkube.TestWorkflowOutput{},
		Workflow:         testworkflowmappers.MapKubeToAPI(initialWorkflow),
		ResolvedWorkflow: testworkflowmappers.MapKubeToAPI(resolvedWorkflow),
	}
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

	e.sendRunWorkflowTelemetry(ctx, &workflow)

	// Start to control the results
	go func() {
		err = e.Control(context.Background(), &execution)
		if err != nil {
			e.handleFatalError(&execution, err, time.Time{})
			return
		}
	}()

	return execution, nil
}

func (e *executor) sendRunWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow) {
	if workflow == nil {
		log.DefaultLogger.Debug("empty workflow passed to telemetry event")
		return
	}
	telemetryEnabled, err := e.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}

	out, err := telemetry.SendRunWorkflowEvent("testkube_api_run_test_workflow", telemetry.RunWorkflowParams{
		RunParams: telemetry.RunParams{
			AppVersion: version.Version,
			DataSource: testworkflowstcl.GetDataSource(workflow.Spec.Content),
			Host:       testworkflowstcl.GetHostname(),
			ClusterID:  testworkflowstcl.GetClusterID(ctx, e.configMap),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(len(workflow.Spec.Setup) + len(workflow.Spec.Steps) + len(workflow.Spec.After)),
			TestWorkflowImage:        testworkflowstcl.GetImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed: testworkflowstcl.HasWorkflowStepLike(workflow.Spec, testworkflowstcl.HasArtifacts),
			TestWorkflowKubeshopGitURI: testworkflowstcl.IsKubeshopGitURI(workflow.Spec.Content) ||
				testworkflowstcl.HasWorkflowStepLike(workflow.Spec, testworkflowstcl.HasKubeshopGitURI),
		},
	})

	if err != nil {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event", "output", out)
	}
}
