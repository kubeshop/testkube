package testworkflowexecutor

import (
	"fmt"
	"maps"
	"strings"
	"time"

	errors2 "github.com/go-errors/errors"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/expressions"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

// Handling sensitive data

const (
	intermediateSensitiveDataFn = "interSensitiveData"
)

type IntermediateExecutionSensitiveData struct {
	Data map[string]string
	ts   time.Time
}

func NewIntermediateExecutionSensitiveData() *IntermediateExecutionSensitiveData {
	return &IntermediateExecutionSensitiveData{
		Data: make(map[string]string),
		ts:   time.Now(),
	}
}

func (s *IntermediateExecutionSensitiveData) Append(_, value string) (expressions.Expression, error) {
	sensitiveId := primitive.NewObjectIDFromTimestamp(s.ts).Hex()
	s.Data[sensitiveId] = value
	return expressions.Compile(fmt.Sprintf(`%s("%s")`, intermediateSensitiveDataFn, sensitiveId))
}

func (s *IntermediateExecutionSensitiveData) Clone() *IntermediateExecutionSensitiveData {
	return &IntermediateExecutionSensitiveData{
		Data: maps.Clone(s.Data),
		ts:   s.ts,
	}
}

type IntermediateExecution struct {
	cr            *testworkflowsv1.TestWorkflow
	dirty         bool
	execution     *testkube.TestWorkflowExecution
	sensitiveData *IntermediateExecutionSensitiveData
	prepended     []string
	tags          map[string]string
	variables     map[string]string
}

// Handling different execution state

func NewIntermediateExecution() *IntermediateExecution {
	return &IntermediateExecution{
		tags:      make(map[string]string),
		variables: make(map[string]string),
		dirty:     true,
		execution: &testkube.TestWorkflowExecution{
			Signature: []testkube.TestWorkflowSignature{},
			Result: &testkube.TestWorkflowResult{
				Status:          common.Ptr(testkube.QUEUED_TestWorkflowStatus),
				PredictedStatus: common.Ptr(testkube.PASSED_TestWorkflowStatus),
				Initialization: &testkube.TestWorkflowStepResult{
					Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
				},
				Steps: map[string]testkube.TestWorkflowStepResult{},
			},
			Output: []testkube.TestWorkflowOutput{},
			Tags:   map[string]string{},
		},
		sensitiveData: NewIntermediateExecutionSensitiveData(),
	}
}

func (e *IntermediateExecution) SetScheduledAt(t time.Time) *IntermediateExecution {
	e.execution.ScheduledAt = t
	e.execution.StatusAt = t
	if e.execution.Result.IsFinished() {
		// TODO: set others too
		e.execution.Result.FinishedAt = t
		e.execution.Result.Initialization.FinishedAt = t
		e.execution.Result.HealDuration(t)
	}
	return e
}

func (e *IntermediateExecution) AutoGenerateID() *IntermediateExecution {
	e.execution.Id = primitive.NewObjectIDFromTimestamp(time.Now()).Hex()
	return e
}

func (e *IntermediateExecution) ID() string {
	return e.execution.Id
}

func (e *IntermediateExecution) GroupID() string {
	return e.execution.GroupId
}

func (e *IntermediateExecution) SetName(name string) *IntermediateExecution {
	e.execution.Name = name
	return e
}

func (e *IntermediateExecution) SetVariables(variables map[string]string) *IntermediateExecution {
	if e.execution.Runtime == nil {
		e.execution.Runtime = &testkube.TestWorkflowExecutionRuntime{}
	}
	if e.execution.Runtime.Variables == nil {
		e.execution.Runtime.Variables = make(map[string]string)
	}
	if e.variables == nil {
		e.variables = make(map[string]string)
	}
	for k, v := range variables {
		e.execution.Runtime.Variables[k] = v
		e.variables[k] = v
	}
	return e
}

func (e *IntermediateExecution) SetGroupID(groupID string) *IntermediateExecution {
	if groupID == "" {
		e.execution.GroupId = e.execution.Id
	} else {
		e.execution.GroupId = groupID
	}
	return e
}

func (e *IntermediateExecution) AppendTags(tags map[string]string) *IntermediateExecution {
	e.dirty = true
	maps.Copy(e.tags, tags)
	return e
}

func (e *IntermediateExecution) SetKubernetesObjectName(name string) *IntermediateExecution {
	e.execution.TestWorkflowExecutionName = name
	return e
}

func (e *IntermediateExecution) SetDisabledWebhooks(disabled bool) *IntermediateExecution {
	e.execution.DisableWebhooks = disabled
	return e
}

func (e *IntermediateExecution) SetRunningContext(runningContext *testkube.TestWorkflowRunningContext) *IntermediateExecution {
	e.execution.RunningContext = runningContext
	return e
}

func (e *IntermediateExecution) WorkflowName() string {
	if e.cr == nil {
		panic("workflow not set yet")
	}
	return e.cr.Name
}

func (e *IntermediateExecution) Name() string {
	return e.execution.Name
}

func (e *IntermediateExecution) Execution() *testkube.TestWorkflowExecution {
	if e.cr == nil {
		panic("workflow not set yet")
	}
	if e.dirty {
		e.dirty = false
		e.execution.ResolvedWorkflow = testworkflowmappers.MapKubeToAPI(e.cr)
		e.execution.Tags = make(map[string]string)
		if e.cr.Spec.Execution != nil {
			// TODO: Should resolve the expressions? (`{{"{{"}}` becomes `{{`)
			maps.Copy(e.execution.Tags, e.cr.Spec.Execution.Tags)
		}
		maps.Copy(e.execution.Tags, e.tags)
	}
	return e.execution
}

func (e *IntermediateExecution) PrependTemplate(name string) *IntermediateExecution {
	if name == "" {
		return e
	}
	if e.cr == nil {
		e.prepended = append(e.prepended, name)
		return e
	}
	e.dirty = true
	testworkflowresolver.AddGlobalTemplateRef(e.cr, testworkflowsv1.TemplateRef{
		Name: testworkflowresolver.GetDisplayTemplateName(name),
	})
	return e
}

func (e *IntermediateExecution) SensitiveData() map[string]string {
	return e.sensitiveData.Data
}

func (e *IntermediateExecution) simplifyCr() error {
	crMachine := testworkflowconfig.CreateWorkflowMachine(&testworkflowconfig.WorkflowConfig{Name: e.cr.Name, Labels: e.cr.Labels})
	err1 := expressions.Simplify(e.cr, crMachine)
	err2 := expressions.SimplifyForce(&e.sensitiveData.Data, crMachine)
	err := errors2.Join(err1, err2)
	e.dirty = true
	if err != nil {
		return err
	}
	return nil
}

func (e *IntermediateExecution) ApplyDynamicConfig(config map[string]string) error {
	if e.cr == nil {
		panic("workflow not set yet")
	}
	e.dirty = true
	_, err := testworkflowresolver.ApplyWorkflowConfig(e.cr, testworkflowmappers.MapConfigValueAPIToKube(config), e.sensitiveData.Append)
	if err != nil {
		return err
	}
	return e.simplifyCr()
}

func (e *IntermediateExecution) ApplyConfig(config map[string]string) error {
	dynamicConfig := make(map[string]string)
	for k, v := range config {
		dynamicConfig[k] = expressions.NewStringValue(v).Template()
	}
	return e.ApplyDynamicConfig(dynamicConfig)
}

func (e *IntermediateExecution) ApplyTemplates(templates map[string]*testkube.TestWorkflowTemplate) error {
	if e.cr == nil {
		panic("workflow not set yet")
	}
	e.dirty = true
	// TODO: apply CRDs directly?
	crTemplates := make(map[string]*testworkflowsv1.TestWorkflowTemplate, len(templates))
	for k, v := range templates {
		crTemplates[k] = testworkflowmappers.MapTemplateAPIToKube(v)
	}
	err := testworkflowresolver.ApplyTemplates(e.cr, crTemplates, e.sensitiveData.Append)
	if err != nil {
		return err
	}
	return e.simplifyCr()
}

func (e *IntermediateExecution) TemplateNames() map[string]struct{} {
	if e.cr == nil {
		panic("workflow not set yet")
	}
	return testworkflowresolver.ListTemplates(e.cr)
}

func (e *IntermediateExecution) Resolve(organizationId, organizationSlug, environmentId, environmentSlug string, parentExecutionIds []string, debug bool) error {
	if e.cr == nil {
		panic("workflow not set yet")
	}
	if e.execution.Id == "" || e.execution.GroupId == "" || e.execution.Name == "" || e.execution.Number == 0 {
		return errors.New("execution is not ready yet")
	}

	if len(e.variables) > 0 {
		if e.cr.Spec.Container == nil {
			e.cr.Spec.Container = &testworkflowsv1.ContainerConfig{}
		}

		for k, v := range e.variables {
			e.cr.Spec.Container.Env = append(e.cr.Spec.Container.Env, testworkflowsv1.EnvVar{
				EnvVar: corev1.EnvVar{
					Name:  k,
					Value: expressions.NewStringValue(v).Template(),
				},
			})
		}
		e.dirty = true
	}

	executionMachine := testworkflowconfig.CreateExecutionMachine(&testworkflowconfig.ExecutionConfig{
		Id:               e.execution.Id,
		GroupId:          e.execution.GroupId,
		Name:             e.execution.Name,
		Number:           e.execution.Number,
		ScheduledAt:      e.execution.ScheduledAt,
		DisableWebhooks:  e.execution.DisableWebhooks,
		Debug:            debug,
		OrganizationId:   organizationId,
		OrganizationSlug: organizationSlug,
		EnvironmentId:    environmentId,
		EnvironmentSlug:  environmentSlug,
		ParentIds:        strings.Join(parentExecutionIds, "/"),
	})
	resourceMachine := testworkflowconfig.CreateResourceMachine(&testworkflowconfig.ResourceConfig{
		Id:     e.execution.Id,
		RootId: e.execution.Id,
	})
	crMachine := testworkflowconfig.CreateWorkflowMachine(&testworkflowconfig.WorkflowConfig{
		Name:   e.cr.Name,
		Labels: e.cr.Labels,
	})
	err1 := expressions.Simplify(e.cr, crMachine, resourceMachine, executionMachine)
	err2 := expressions.SimplifyForce(&e.sensitiveData.Data, crMachine, resourceMachine, executionMachine)
	err := errors2.Join(err1, err2)
	e.dirty = true
	if err != nil {
		return err
	}
	return nil
}

func (e *IntermediateExecution) SetWorkflow(workflow *testworkflowsv1.TestWorkflow) *IntermediateExecution {
	e.cr = workflow.DeepCopy()
	e.dirty = true
	e.execution.Workflow = testworkflowmappers.MapKubeToAPI(e.cr)
	for _, tpl := range e.prepended {
		testworkflowresolver.AddGlobalTemplateRef(e.cr, testworkflowsv1.TemplateRef{
			Name: testworkflowresolver.GetDisplayTemplateName(tpl),
		})
	}
	e.prepended = nil
	return e
}

func (e *IntermediateExecution) SetTarget(target testkube.ExecutionTarget) *IntermediateExecution {
	e.execution.RunnerTarget = &target
	return e
}

func (e *IntermediateExecution) SetOriginalTarget(target testkube.ExecutionTarget) *IntermediateExecution {
	e.execution.RunnerOriginalTarget = &target
	return e
}

func (e *IntermediateExecution) SetSequenceNumber(number int32) *IntermediateExecution {
	e.execution.Number = number
	return e
}

func (e *IntermediateExecution) SequenceNumber() int32 {
	return e.execution.Number
}

func (e *IntermediateExecution) SetError(header string, err error) *IntermediateExecution {
	// Keep only the 1st error
	if !e.execution.Result.IsFinished() {
		e.execution.InitializationError(header, err)
	}
	return e
}

func (e *IntermediateExecution) RewriteSensitiveDataCall(handler func(name string) (expressions.Expression, error)) error {
	e.dirty = true
	return expressions.Simplify(&e.cr, expressions.NewMachine().RegisterFunction(intermediateSensitiveDataFn, func(values ...expressions.StaticValue) (interface{}, bool, error) {
		if len(values) != 1 {
			return nil, true, fmt.Errorf(`"%s" function expects 1 argument, %d provided`, intermediateSensitiveDataFn, len(values))
		}
		localCredentialName, _ := values[0].StringValue()
		expr, err := handler(localCredentialName)
		return expr, true, err
	}))
}

func (e *IntermediateExecution) StoreConfig(config map[string]string) *IntermediateExecution {
	params := make(map[string]testkube.TestWorkflowExecutionConfigValue)
	for k, v := range config {
		if s, ok := e.cr.Spec.Config[k]; ok && !s.Sensitive {
			params[k] = testkube.TestWorkflowExecutionConfigValue{Value: v}
		}
	}
	e.execution.ConfigParams = params
	return e
}

func (e *IntermediateExecution) Finished() bool {
	return e.execution.Result.IsFinished()
}

func (e *IntermediateExecution) Clone() *IntermediateExecution {
	return &IntermediateExecution{
		cr:            e.cr.DeepCopy(),
		dirty:         e.dirty,
		tags:          maps.Clone(e.tags),
		execution:     e.execution.Clone(),
		prepended:     e.prepended,
		sensitiveData: e.sensitiveData.Clone(),
		variables:     maps.Clone(e.variables),
	}
}
