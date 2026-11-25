package testkube

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestWorkflowExecution_InitializationError(t *testing.T) {
	tests := []struct {
		name             string
		header           string
		reason           string
		err              error
		wantErrorMessage string
	}{
		{
			name:             "puts the plain header in front of the error",
			header:           "Cannot fetch the templates of the workflow",
			reason:           string(StopReasonTemplateMissing),
			err:              errors.New("template not found"),
			wantErrorMessage: "Cannot fetch the templates of the workflow: template not found",
		},
		{
			name:             "stores only the error when the header is empty",
			header:           "",
			reason:           string(StartReasonImagePullFailed),
			err:              errors.New("image not found"),
			wantErrorMessage: "image not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TestWorkflowExecution{
				Result: &TestWorkflowResult{
					Initialization: &TestWorkflowStepResult{},
					Steps:          map[string]TestWorkflowStepResult{},
				},
			}

			e.InitializationError(tt.header, tt.reason, tt.err)

			assert.Equal(t, tt.wantErrorMessage, e.Result.Initialization.ErrorMessage)
			assert.Equal(t, tt.reason, e.Result.Initialization.ErrorReason)
		})
	}
}

func TestTestWorkflowExecution_AgentSteps(t *testing.T) {
	withSpec := func(spec TestWorkflowSpec) *TestWorkflowExecution {
		return &TestWorkflowExecution{ResolvedWorkflow: &TestWorkflow{Spec: &spec}}
	}

	tests := []struct {
		name      string
		execution *TestWorkflowExecution
		want      []string
	}{
		{name: "nil execution", execution: nil},
		{name: "no resolved workflow", execution: &TestWorkflowExecution{}},
		{name: "no spec", execution: &TestWorkflowExecution{ResolvedWorkflow: &TestWorkflow{}}},
		{
			name:      "no agent step",
			execution: withSpec(TestWorkflowSpec{Steps: []TestWorkflowStep{{Shell: "true"}}}),
		},
		{
			name: "setup, steps and after, in that order",
			execution: withSpec(TestWorkflowSpec{
				Setup: []TestWorkflowStep{{Agent: agentNamed("setup")}},
				Steps: []TestWorkflowStep{{Agent: agentNamed("steps")}},
				After: []TestWorkflowStep{{Agent: agentNamed("after")}},
			}),
			want: []string{"setup", "steps", "after"},
		},
		{
			name: "the same AI Agent in two steps is in the result two times",
			execution: withSpec(TestWorkflowSpec{
				Steps: []TestWorkflowStep{{Agent: agentNamed("triage")}, {Agent: agentNamed("triage")}},
			}),
			want: []string{"triage", "triage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, agentNames(tt.execution.AgentSteps()))
		})
	}
}

func agentNamed(name string) *TestWorkflowStepAgent {
	return &TestWorkflowStepAgent{Name: name}
}

// agentNames returns the names of the AI Agents, and nil for no agent step.
func agentNames(agents []TestWorkflowStepAgent) []string {
	var names []string
	for _, a := range agents {
		names = append(names, a.Name)
	}
	return names
}
