package testworkflowexecutor

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/errgroup"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testworkflowsv1client "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/log"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
	runner2 "github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
	localCredentialFnName = "localUserCredentials" // TODO: Random
)

type ExecutionScheduler struct {
	testWorkflowsClient          testworkflowsv1client.Interface
	testWorkflowTemplatesClient  testworkflowsv1client.TestWorkflowTemplatesInterface
	testWorkflowExecutionsClient testworkflowsv1client.TestWorkflowExecutionsInterface
	secretManager                secretmanager.SecretManager
	repository                   testworkflow2.Repository
	outputRepository             testworkflow2.OutputRepository
	runner                       runner2.Runner
	globalTemplateName           string
	emitter                      *event.Emitter
}

type ScheduleRequest struct {
	// Test Workflow details
	Name   string            `json:"name,omitempty"`
	Config map[string]string `json:"config,omitempty"`

	// Execution details
	ExecutionName   string            `json:"executionName,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
	DisableWebhooks bool              `json:"disableWebhooks,omitempty"`

	// Kubernetes resource
	TestWorkflowExecutionObjectName string `json:"testWorkflowExecutionObjectName,omitempty"`

	// Deprecated
	RunningContext *testkube.TestWorkflowRunningContext `json:"runningContext,omitempty"`
	// Deprecated
	ParentExecutionIds []string `json:"parentExecutionIds,omitempty"`
}

type PreparedExecution struct {
	Execution     testkube.TestWorkflowExecution
	SensitiveData map[string]string
}

func NewExecutionScheduler(
	testWorkflowsClient testworkflowsv1client.Interface,
	testWorkflowTemplatesClient testworkflowsv1client.TestWorkflowTemplatesInterface,
	testWorkflowExecutionsClient testworkflowsv1client.TestWorkflowExecutionsInterface,
	secretManager secretmanager.SecretManager,
	repository testworkflow2.Repository,
	outputRepository testworkflow2.OutputRepository,
	runner runner2.Runner,
	globalTemplateName string,
	emitter *event.Emitter,
) *ExecutionScheduler {
	return &ExecutionScheduler{
		testWorkflowsClient:          testWorkflowsClient,
		testWorkflowTemplatesClient:  testWorkflowTemplatesClient,
		testWorkflowExecutionsClient: testWorkflowExecutionsClient,
		secretManager:                secretManager,
		repository:                   repository,
		outputRepository:             outputRepository,
		runner:                       runner,
		globalTemplateName:           globalTemplateName,
		emitter:                      emitter,
	}
}

// TODO: Consider if we shouldn't make name unique across all TestWorkflows
func (s *ExecutionScheduler) isExecutionNameReserved(ctx context.Context, name, workflowName string) (bool, error) {
	// TODO: Detect errors other than 404?
	next, _ := s.repository.GetByNameAndTestWorkflow(ctx, name, workflowName)
	if next.Name == name {
		return true, nil
	}
	return false, nil
}

func (s *ExecutionScheduler) PrepareExecutionBase(ctx context.Context, data ScheduleRequest) (*PreparedExecution, error) {
	// -----=====[ 01 ]=====[ Build initial data ]=====-------
	now := time.Now().UTC()
	groupId := primitive.NewObjectIDFromTimestamp(now).Hex()

	// -----=====[ 02 ]=====[ Treat config provided by user as literal ]=====------- TODO: should be this way?
	config := make(map[string]string)
	for k, v := range data.Config {
		config[k] = expressions.NewStringValue(v).Template()
	}

	// -----=====[ 03 ]=====[ Prepare store for the sensitive data ]=====-------
	sensitiveData := make(map[string]string)
	sensitiveDataAppend := func(key, value string) (expressions.Expression, error) {
		sensitiveId := primitive.NewObjectIDFromTimestamp(now).Hex()
		sensitiveData[sensitiveId] = value
		return expressions.Compile(fmt.Sprintf(`%s("%s")`, localCredentialFnName, sensitiveId))
	}

	// -----=====[ 04 ]=====[ Get the TestWorkflow from the storage ]=====-------
	workflow, err := s.testWorkflowsClient.Get(data.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get workflow '%s'", data.Name)
	}

	// -----=====[ 05 ]=====[ Keep the initial workflow spec ]=====-------
	initialWorkflow := workflow.DeepCopy()
	initialWorkflowApi := testworkflowmappers.MapKubeToAPI(initialWorkflow)
	workflowMachine := testworkflowconfig.CreateWorkflowMachine(&testworkflowconfig.WorkflowConfig{Name: workflow.Name, Labels: workflow.Labels})

	// -----=====[ XX ]=====[ Instantiate the execution base ]=====-------
	base := &testkube.TestWorkflowExecution{
		GroupId:     groupId,
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
		TestWorkflowExecutionName: data.TestWorkflowExecutionObjectName,
		DisableWebhooks:           data.DisableWebhooks,
		Tags:                      map[string]string{},
		RunningContext:            data.RunningContext,
	}

	// Inject configuration
	if testworkflows.CountMapBytes(data.Config) < ConfigSizeLimit {
		storeConfig := true
		schema := workflow.Spec.Config
		for k := range data.Config {
			if s, ok := schema[k]; ok && s.Sensitive {
				storeConfig = false
			}
		}
		if storeConfig {
			base.Config = data.Config
		}
	}

	// -----=====[ 06 ]=====[ Auto-inject the global template ]=====-------
	if s.globalTemplateName != "" {
		testworkflowresolver.AddGlobalTemplateRef(workflow, testworkflowsv1.TemplateRef{
			Name: testworkflowresolver.GetDisplayTemplateName(s.globalTemplateName),
		})
	}

	// -----=====[ 09 ]=====[ Simplify the Test Workflow initially ]=====-------
	err = expressions.Simplify(&workflow)
	if err != nil {
		base.ResolvedWorkflow = common.Ptr(testworkflowmappers.MapTestWorkflowKubeToAPI(*workflow))
		base.InitializationError("Cannot process Test Workflow specification", err)
		return &PreparedExecution{SensitiveData: sensitiveData, Execution: *base}, nil
	}

	// -----=====[ 10 ]=====[ Fetch all required templates ]=====-------
	tpls := testworkflowresolver.ListTemplates(workflow)
	tplsMap := make(map[string]testworkflowsv1.TestWorkflowTemplate, len(tpls))
	var tplsMu sync.Mutex
	var g errgroup.Group
	for tplName := range tpls {
		func(tplName string) {
			g.Go(func() error {
				tpl, err := s.testWorkflowTemplatesClient.Get(tplName)
				if err != nil {
					return errors.Wrap(err, testworkflowresolver.GetDisplayTemplateName(tplName))
				}
				tplsMu.Lock()
				defer tplsMu.Unlock()
				tplsMap[tplName] = *tpl
				return nil
			})
		}(tplName)
	}
	err = g.Wait()
	if err != nil {
		base.ResolvedWorkflow = common.Ptr(testworkflowmappers.MapTestWorkflowKubeToAPI(*workflow))
		base.InitializationError("Cannot fetch required Test Workflow Templates", err)
		return &PreparedExecution{SensitiveData: sensitiveData, Execution: *base}, nil
	}

	// -----=====[ 11 ]=====[ Inline the Test Workflow configuration ]=====-------
	_, err = testworkflowresolver.ApplyWorkflowConfig(workflow, testworkflowmappers.MapConfigValueAPIToKube(config), sensitiveDataAppend)
	if err != nil {
		base.ResolvedWorkflow = common.Ptr(testworkflowmappers.MapTestWorkflowKubeToAPI(*workflow))
		base.InitializationError("Cannot inline Test Workflow configuration", err)
		return &PreparedExecution{SensitiveData: sensitiveData, Execution: *base}, nil
	}

	// -----=====[ 12 ]=====[ Resolve the Test Workflow with templates ]=====-------
	err = testworkflowresolver.ApplyTemplates(workflow, tplsMap, sensitiveDataAppend)
	if err != nil {
		base.ResolvedWorkflow = common.Ptr(testworkflowmappers.MapTestWorkflowKubeToAPI(*workflow))
		base.InitializationError("Cannot inline Test Workflow templates", err)
		return &PreparedExecution{SensitiveData: sensitiveData, Execution: *base}, nil
	}

	// -----=====[ 13 ]=====[ Resolve common values ]=====-------
	err = expressions.Simplify(&workflow, workflowMachine)
	if err != nil {
		base.ResolvedWorkflow = common.Ptr(testworkflowmappers.MapTestWorkflowKubeToAPI(*workflow))
		base.InitializationError("Cannot inline Test Workflow templates", err)
		return &PreparedExecution{SensitiveData: sensitiveData, Execution: *base}, nil
	}

	base.ResolvedWorkflow = common.Ptr(testworkflowmappers.MapTestWorkflowKubeToAPI(*workflow))

	return &PreparedExecution{SensitiveData: sensitiveData, Execution: *base}, nil
}

func (s *ExecutionScheduler) PrepareExecutions(ctx context.Context, base *PreparedExecution, organizationId, environmentId string, data ScheduleRequest) ([]PreparedExecution, error) {
	baseWorkflow := testworkflowmappers.MapTestWorkflowAPIToKube(*base.Execution.ResolvedWorkflow)
	workflowMachine := testworkflowconfig.CreateWorkflowMachine(&testworkflowconfig.WorkflowConfig{Name: baseWorkflow.Name, Labels: baseWorkflow.Labels})

	// TODO: when multiple executions will be scheduled with ExecutionName,
	//       fail immediately or add "-<N>" suffix by default.

	// -----=====[ 07 ]=====[ Reserve the execution ]=====-------
	executionId := primitive.NewObjectIDFromTimestamp(time.Now()).Hex()
	executionName := data.ExecutionName

	var nameReserved *bool

	// Early check if the name is already provided (to avoid incrementing sequence number when we won't succeed anyway)
	if executionName != "" {
		reserved, err := s.isExecutionNameReserved(ctx, executionName, baseWorkflow.Name)
		if err != nil {
			return nil, errors.Wrap(err, "checking for unique name")
		}
		if reserved {
			return nil, errors.New("execution name already exists")
		}
		nameReserved = &reserved
	}

	// Load execution identifier data
	executionNumber, err := s.repository.GetNextExecutionNumber(context.Background(), baseWorkflow.Name)
	if err != nil {
		return nil, errors.Wrap(err, "registering next execution sequence number")
	}
	if executionName == "" {
		executionName = fmt.Sprintf("%s-%d", baseWorkflow.Name, executionNumber)
	}

	// Ensure the execution name is unique
	if nameReserved == nil {
		reserved, err := s.isExecutionNameReserved(ctx, executionName, baseWorkflow.Name)
		if err != nil {
			return nil, errors.Wrap(err, "checking for unique name")
		}
		if reserved {
			return nil, errors.New("execution name already exists")
		}
	}

	// -----=====[ 08 ]=====[ Build the list of executions ]=====-------
	executionOne := base.Execution
	executionOne.Id = executionId
	executionOne.Name = executionName
	executionOne.Number = executionNumber
	executions := []PreparedExecution{{Execution: executionOne}}

	// -----=====[ 14 ]=====[ Resolve and Queue all the executions ]=====-------
	for i := range executions {
		// Ignore if it's already considered finished
		if executions[i].Execution.Result.IsFinished() {
			continue
		}

		// Resolve execution-specific values
		executionWorkflow := baseWorkflow.DeepCopy()
		executionMachine := testworkflowconfig.CreateExecutionMachine(&testworkflowconfig.ExecutionConfig{
			Id:              executions[i].Execution.Id,
			GroupId:         executions[i].Execution.GroupId,
			Name:            executions[i].Execution.Name,
			Number:          executions[i].Execution.Number,
			ScheduledAt:     executions[i].Execution.ScheduledAt,
			DisableWebhooks: executions[i].Execution.DisableWebhooks,
			Debug:           false,
			OrganizationId:  organizationId,
			EnvironmentId:   environmentId,
			ParentIds:       strings.Join(data.ParentExecutionIds, "/"),
		})
		resourceMachine := testworkflowconfig.CreateResourceMachine(&testworkflowconfig.ResourceConfig{
			Id:     executions[i].Execution.Id,
			RootId: executions[i].Execution.Id,
		})

		// Apply the execution-specific data
		err = expressions.Simplify(&executionWorkflow, workflowMachine, executionMachine, resourceMachine)
		if err != nil {
			(&executions[i].Execution).InitializationError("Cannot process Test Workflow specification", err)
			continue
		}
		executions[i].Execution.ResolvedWorkflow = testworkflowmappers.MapKubeToAPI(executionWorkflow)

		// Resolve sensitive data
		executions[i].SensitiveData = make(map[string]string, len(base.SensitiveData))
		for k := range base.SensitiveData {
			expr, err := expressions.CompileAndResolveTemplate(base.SensitiveData[k], workflowMachine, executionMachine, resourceMachine)
			if err != nil {
				(&executions[i].Execution).InitializationError("Cannot process Test Workflow sensitive data", err)
				continue
			}
			executions[i].SensitiveData[k] = expr.Template()
		}
	}

	// Load tags for the execution
	for i := range executions {
		executions[i].Execution.Tags = map[string]string{}
		if executions[i].Execution.ResolvedWorkflow.Spec.Execution != nil {
			maps.Copy(executions[i].Execution.Tags, executions[i].Execution.ResolvedWorkflow.Spec.Execution.Tags)
		}
		maps.Copy(executions[i].Execution.Tags, data.Tags)
	}

	return executions, nil
}

func retry(count int, delayBase time.Duration, fn func() error) (err error) {
	for i := 0; i < count; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * delayBase)
	}
	return err
}

func (s *ExecutionScheduler) saveEmptyLogs(execution *testkube.TestWorkflowExecution) {
	err := retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
		return s.outputRepository.SaveLog(context.Background(), execution.Id, execution.Workflow.Name, bytes.NewReader(nil))
	})
	if err != nil {
		log.DefaultLogger.Errorw("failed to save empty log", "executionId", execution.Id, "error", err)
	}
}

func (s *ExecutionScheduler) saveExecutionInKubernetes(execution *testkube.TestWorkflowExecution) error {
	// TODO: retry?
	// TODO: Move it as a side effect in the Agent (CRD Sync)
	if execution.TestWorkflowExecutionName != "" {
		cr, err := s.testWorkflowExecutionsClient.Get(execution.TestWorkflowExecutionName)
		if k8serrors.IsNotFound(err) {
			// TODO: think if it should be inserted?
			return nil
		}
		if err != nil {
			return err
		}
		cr.Status = testworkflowmappers.MapTestWorkflowExecutionStatusAPIToKube(execution, cr.Generation)
		return s.testWorkflowExecutionsClient.UpdateStatus(cr)
	}
	return nil
}

func (s *ExecutionScheduler) insert(execution *testkube.TestWorkflowExecution) error {
	var g errgroup.Group
	g.Go(func() error {
		return retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
			return s.repository.Insert(context.Background(), *execution)
		})
	})
	if execution.TestWorkflowExecutionName != "" {
		g.Go(func() error {
			return s.saveExecutionInKubernetes(execution)
		})
	}
	return g.Wait()
}

func (s *ExecutionScheduler) update(execution *testkube.TestWorkflowExecution) error {
	var g errgroup.Group
	g.Go(func() error {
		return retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
			return s.repository.Insert(context.Background(), *execution)
		})
	})
	if execution.TestWorkflowExecutionName != "" {
		g.Go(func() error {
			return s.saveExecutionInKubernetes(execution)
		})
	}
	return g.Wait()
}

func (s *ExecutionScheduler) DoOne(controlPlaneConfig testworkflowconfig.ControlPlaneConfig, organizationId, environmentId string, parentExecutionIds []string, exec PreparedExecution) (testkube.TestWorkflowExecution, error) {
	// Prepare the sensitive data TODO: Use Credentials when possible
	secretsBatch := s.secretManager.Batch("twe-", exec.Execution.Id).ForceEnable()
	credentialExpressions := map[string]expressions.Expression{}
	for k, v := range exec.SensitiveData {
		envVarSource, err := secretsBatch.Append(k, v)
		if err != nil {
			return exec.Execution, errors.Wrapf(err, "cannot resolve workflow '%s'", exec.Execution.Workflow.Name)
		}
		credentialExpressions[k] = expressions.MustCompile(fmt.Sprintf(`secret("%s","%s",true)`, envVarSource.SecretKeyRef.Name, envVarSource.SecretKeyRef.Key))
	}
	secrets := secretsBatch.Get()
	//for j := range secrets {
	//	testworkflowprocessor.AnnotateControlledBy(&secrets[j], exec.Execution.Id, exec.Execution.Id)
	//}
	err := expressions.Simplify(&exec.Execution.ResolvedWorkflow, expressions.NewMachine().RegisterFunction(localCredentialFnName, func(values ...expressions.StaticValue) (interface{}, bool, error) {
		if len(values) != 1 {
			return nil, true, fmt.Errorf(`"%s" function expects 1 argument, %d provided`, localCredentialFnName, len(values))
		}
		localCredentialName, _ := values[0].StringValue()
		if expr, ok := credentialExpressions[localCredentialName]; ok {
			return expr, true, nil
		}
		return nil, true, fmt.Errorf(`"%s" local credential not found`, localCredentialName)
	}))
	if err != nil {
		// TODO: delete the credentials left-overs
		// TODO(init): fail only this execution
		return exec.Execution, errors.Wrapf(err, "cannot resolve workflow '%s'", exec.Execution.Workflow.Name)
	}

	// Store the sensitive data in the cluster TODO: Use credentials when possible
	secretsMap := make(map[string]map[string]string, len(secrets))
	for j := range secrets {
		secretsMap[secrets[j].Name] = secrets[j].StringData
	}

	// Insert the execution
	err = s.insert(&exec.Execution)
	if err != nil {
		// TODO: delete the credentials left-overs
		// TODO: don't fail immediately (try creating other executions too)
		return exec.Execution, errors.Wrapf(err, "cannot insert execution '%s' result for workflow '%s'", exec.Execution.Id, exec.Execution.Workflow.Name)
	}
	s.emitter.Notify(testkube.NewEventQueueTestWorkflow(&exec.Execution))

	// Finish early if it's immediately known to finish
	if exec.Execution.Result.IsFinished() {
		//s.emitter.Notify(testkube.NewEventStartTestWorkflow(&exec.Execution)) // TODO: Consider
		if exec.Execution.Result.IsAborted() {
			s.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(&exec.Execution))
		} else if exec.Execution.Result.IsFailed() {
			s.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(&exec.Execution))
		} else {
			s.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(&exec.Execution))
		}
		s.saveEmptyLogs(&exec.Execution)
		return exec.Execution, nil
	}

	// Start the execution
	result, err := s.runner.Execute(executionworkertypes.ExecuteRequest{
		Execution: testworkflowconfig.ExecutionConfig{
			Id:              exec.Execution.Id,
			GroupId:         exec.Execution.GroupId,
			Name:            exec.Execution.Name,
			Number:          exec.Execution.Number,
			ScheduledAt:     exec.Execution.ScheduledAt,
			DisableWebhooks: exec.Execution.DisableWebhooks,
			Debug:           false,
			OrganizationId:  organizationId,
			EnvironmentId:   environmentId,
			ParentIds:       strings.Join(parentExecutionIds, "/"),
		},
		Secrets:      secretsMap,
		Workflow:     testworkflowmappers.MapTestWorkflowAPIToKube(*exec.Execution.ResolvedWorkflow),
		ControlPlane: controlPlaneConfig,
	})

	// TODO: define "revoke" error by runner (?)
	if err != nil {
		exec.Execution.InitializationError("Failed to run execution", err)
		var g errgroup.Group
		g.Go(func() error {
			return retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
				return s.repository.Update(context.Background(), exec.Execution)
			})
		})
		if exec.Execution.TestWorkflowExecutionName != "" {
			// TODO: Move it as a side effect in the Agent
			g.Go(func() error {
				cr, err := s.testWorkflowExecutionsClient.Get(exec.Execution.TestWorkflowExecutionName)
				if err != nil {
					return err
				}
				cr.Status = testworkflowmappers.MapTestWorkflowExecutionStatusAPIToKube(&exec.Execution, cr.Generation)
				return s.testWorkflowExecutionsClient.UpdateStatus(cr)
			})
		}
		if err != nil {
			return exec.Execution, errors.Wrap(err, "failed to update the execution")
		}

		//s.emitter.Notify(testkube.NewEventStartTestWorkflow(&exec.Execution)) // TODO: Consider
		s.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(&exec.Execution))
		s.saveEmptyLogs(&exec.Execution)

		return exec.Execution, nil
	}

	// Inform about execution start TODO: Consider
	//s.emitter.Notify(testkube.NewEventStartTestWorkflow(&exec.Execution))

	// Apply the signature
	// TODO: it should be likely scheduled from the Runner,
	//       otherwise there may be race condition between Runner and that Scheduler.
	//       Alternatively, it could check for `signature` existence in the DB.
	exec.Execution.Namespace = result.Namespace
	exec.Execution.Signature = result.Signature
	exec.Execution.Result.Steps = stage.MapSignatureListToStepResults(stage.MapSignatureList(result.Signature))
	err = retry(SaveResultRetryMaxAttempts, SaveResultRetryBaseDelay, func() error {
		return s.repository.Update(context.Background(), exec.Execution)
	})
	if err != nil {
		return exec.Execution, errors.Wrap(err, "failed to save the signature")
	}

	return exec.Execution, nil
}

// TODO: Ensure there are no metrics required and deleted
// TODO: Should it return channel instead (?)
func (s *ExecutionScheduler) Do(ctx context.Context, dashboardURI, organizationId, environmentId string, data ScheduleRequest) (executions []testkube.TestWorkflowExecution, err error) {
	base, err := s.PrepareExecutionBase(ctx, data)
	if err != nil {
		return nil, err
	}

	preparedExecutions, err := s.PrepareExecutions(ctx, base, organizationId, environmentId, data)
	if err != nil {
		return nil, err
	}

	// TODO: Delete it, as it won't happen for OSS
	if organizationId != "" && environmentId != "" && dashboardURI == "" {
		cloudUiUrl := os.Getenv("TESTKUBE_PRO_UI_URL")
		dashboardURI = fmt.Sprintf("%s/organization/%s/environment/%s/dashboard", cloudUiUrl, organizationId, environmentId)
	}
	controlPlaneConfig := testworkflowconfig.ControlPlaneConfig{
		DashboardUrl:   dashboardURI,
		CDEventsTarget: os.Getenv("CDEVENTS_TARGET"),
	}

	executions = make([]testkube.TestWorkflowExecution, len(preparedExecutions))
	for i := range preparedExecutions {
		executions[i], err = s.DoOne(controlPlaneConfig, organizationId, environmentId, data.ParentExecutionIds, preparedExecutions[i])
		if err != nil && !executions[i].Result.IsFinished() {
			// TODO: apply internal error to the execution
		}
	}

	return executions, nil
}
