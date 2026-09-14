package executiondata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestFromExecution(t *testing.T) {
	tests := []struct {
		name      string
		execution *testkube.TestWorkflowExecution
		want      Execution
	}{
		{
			name:      "leaves the error fields empty when the execution has no result",
			execution: &testkube.TestWorkflowExecution{Id: "exec-1", Name: "wf-1"},
			want:      Execution{Id: "exec-1", Name: "wf-1", Outputs: map[string]string{}},
		},
		{
			name: "fills the initialization message and the step messages keyed by ref",
			execution: &testkube.TestWorkflowExecution{
				Id:       "exec-1",
				Name:     "wf-1",
				Workflow: &testkube.TestWorkflow{Name: "wf"},
				Result: &testkube.TestWorkflowResult{
					Status:         common.Ptr(testkube.ABORTED_TestWorkflowStatus),
					Initialization: &testkube.TestWorkflowStepResult{ErrorMessage: "the pod cannot be scheduled"},
					Steps: map[string]testkube.TestWorkflowStepResult{
						"rstep1": {ErrorMessage: "the step timed out"},
						"rstep2": {ErrorMessage: ""},
					},
				},
			},
			want: Execution{
				Id:           "exec-1",
				Name:         "wf-1",
				Workflow:     "wf",
				Status:       "aborted",
				Outputs:      map[string]string{},
				ErrorMessage: "the pod cannot be scheduled",
				StepErrors:   map[string]string{"rstep1": "the step timed out"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FromExecution(tt.execution))
		})
	}
}

func TestExecution_AsMap(t *testing.T) {
	tests := []struct {
		name           string
		execution      Execution
		wantMessage    interface{}
		wantStepErrors interface{}
	}{
		{
			name: "exposes the error message and the step errors",
			execution: Execution{
				ErrorMessage: "the pod cannot be scheduled",
				StepErrors:   map[string]string{"rstep1": "the step timed out"},
			},
			wantMessage:    "the pod cannot be scheduled",
			wantStepErrors: map[string]interface{}{"rstep1": "the step timed out"},
		},
		{
			name:           "exposes empty values when the execution has no errors",
			execution:      Execution{},
			wantMessage:    "",
			wantStepErrors: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.execution.AsMap()
			assert.Equal(t, tt.wantMessage, got["errorMessage"])
			assert.Equal(t, tt.wantStepErrors, got["stepErrors"])
		})
	}
}
