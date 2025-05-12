package testkube

import (
	"encoding/json"
	"fmt"

	"github.com/gookit/color"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/utils"
)

type TestWorkflowExecutions []TestWorkflowExecution

func (executions TestWorkflowExecutions) Table() (header []string, output [][]string) {
	header = []string{"Id", "Name", "Test Workflow Name", "Status", "Labels", "Tags"}

	for _, e := range executions {
		status := "unknown"
		if e.Result != nil && e.Result.Status != nil {
			status = string(*e.Result.Status)
		}

		output = append(output, []string{
			e.Id,
			e.Name,
			e.Workflow.Name,
			status,
			MapToString(e.Workflow.Labels),
			MapToString(e.Tags),
		})
	}

	return
}

func (e *TestWorkflowExecution) ConvertDots(fn func(string) string) *TestWorkflowExecution {
	e.Workflow.ConvertDots(fn)
	e.ResolvedWorkflow.ConvertDots(fn)
	if e.Tags != nil {
		e.Tags = convertDotsInMap(e.Tags, fn)
	}
	return e
}

func (e *TestWorkflowExecution) EscapeDots() *TestWorkflowExecution {
	return e.ConvertDots(utils.EscapeDots)
}

func (e *TestWorkflowExecution) UnscapeDots() *TestWorkflowExecution {
	return e.ConvertDots(utils.UnescapeDots)
}

func (e *TestWorkflowExecution) GetNamespace(defaultNamespace string) string {
	if e.Namespace == "" {
		return defaultNamespace
	}
	return e.Namespace
}

func (e *TestWorkflowExecution) ContainsExecuteAction() bool {
	if e == nil {
		return false
	}

	if e.ResolvedWorkflow == nil || e.ResolvedWorkflow.Spec == nil {
		return false
	}

	steps := append(e.ResolvedWorkflow.Spec.Setup, append(e.ResolvedWorkflow.Spec.Steps, e.ResolvedWorkflow.Spec.After...)...)
	for _, step := range steps {
		if step.ContainsExecuteAction() {
			return true
		}
	}

	return false
}

func (e *TestWorkflowExecution) GetTemplateRefs() []TestWorkflowTemplateRef {
	if e == nil {
		return nil
	}

	if e.ResolvedWorkflow == nil || e.ResolvedWorkflow.Spec == nil {
		return nil
	}

	var templateRefs []TestWorkflowTemplateRef
	steps := append(e.ResolvedWorkflow.Spec.Setup, append(e.ResolvedWorkflow.Spec.Steps, e.ResolvedWorkflow.Spec.After...)...)
	for _, step := range steps {
		templateRefs = append(templateRefs, step.GetTemplateRefs()...)
	}

	return templateRefs
}

func (e *TestWorkflowExecution) InitializationError(header string, err error) {
	e.Result.Status = common.Ptr(ABORTED_TestWorkflowStatus)
	e.Result.PredictedStatus = e.Result.Status
	e.Result.FinishedAt = e.ScheduledAt
	e.Result.Initialization.Status = common.Ptr(ABORTED_TestWorkflowStepStatus)
	e.Result.Initialization.FinishedAt = e.ScheduledAt
	e.Result.Initialization.ErrorMessage = err.Error()
	if header != "" {
		e.Result.Initialization.ErrorMessage = fmt.Sprintf("%s\n%s", color.Bold.Render(header), e.Result.Initialization.ErrorMessage)
	}
	for ref, step := range e.Result.Steps {
		step.Status = common.Ptr(SKIPPED_TestWorkflowStepStatus)
		step.FinishedAt = e.ScheduledAt
		e.Result.Steps[ref] = step
	}
	e.Result.HealDuration(e.ScheduledAt)
}

func (e *TestWorkflowExecution) FailedToInitialize() bool {
	return e.Result.Status != nil && *e.Result.Status == ABORTED_TestWorkflowStatus && e.Result.QueuedAt.IsZero()
}

func (e *TestWorkflowExecution) GetParallelStepReference(nameOrReference string) string {
	if e == nil {
		return ""
	}

	for _, signature := range e.Signature {
		ref := signature.GetParallelStepReference(nameOrReference)
		if ref != "" {
			return ref
		}
	}

	return ""
}

func (e *TestWorkflowExecution) Assigned() bool {
	return e.Result.IsFinished() || len(e.Signature) > 0
}

func (e *TestWorkflowExecution) Clone() *TestWorkflowExecution {
	if e == nil {
		return nil
	}
	v, _ := json.Marshal(e)
	result := TestWorkflowExecution{}
	_ = json.Unmarshal(v, &result)
	return &result
}

func findParent(parentRef, ref string, signature []TestWorkflowSignature) string {
	for _, s := range signature {
		if s.Ref == ref {
			return parentRef
		}

		if s.Children != nil {
			parentRef = findParent(s.Ref, ref, s.Children)
			if parentRef != "" {
				return parentRef
			}
		}
	}

	return ""
}

func (e *TestWorkflowExecution) GetParentRef(ref string) string {
	return findParent("", ref, e.Signature)
}
