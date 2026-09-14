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
			name: "fills the initialization message, the step messages, and the step attempts keyed by ref",
			execution: &testkube.TestWorkflowExecution{
				Id:       "exec-1",
				Name:     "wf-1",
				Workflow: &testkube.TestWorkflow{Name: "wf"},
				Result: &testkube.TestWorkflowResult{
					Status:         common.Ptr(testkube.ABORTED_TestWorkflowStatus),
					Initialization: &testkube.TestWorkflowStepResult{ErrorMessage: "the pod cannot be scheduled"},
					Steps: map[string]testkube.TestWorkflowStepResult{
						"rstep1": {ErrorMessage: "the step timed out", Attempts: 3},
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
				StepAttempts: map[string]int64{"rstep1": 3},
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
		wantAttempts   interface{}
	}{
		{
			name: "exposes the error message, the step errors, and the step attempts",
			execution: Execution{
				ErrorMessage: "the pod cannot be scheduled",
				StepErrors:   map[string]string{"rstep1": "the step timed out"},
				StepAttempts: map[string]int64{"rstep1": 3},
			},
			wantMessage:    "the pod cannot be scheduled",
			wantStepErrors: map[string]interface{}{"rstep1": "the step timed out"},
			wantAttempts:   map[string]interface{}{"rstep1": int64(3)},
		},
		{
			name:           "exposes empty values when the execution has no errors and no attempts",
			execution:      Execution{},
			wantMessage:    "",
			wantStepErrors: map[string]interface{}{},
			wantAttempts:   map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.execution.AsMap()
			assert.Equal(t, tt.wantMessage, got["errorMessage"])
			assert.Equal(t, tt.wantStepErrors, got["stepErrors"])
			assert.Equal(t, tt.wantAttempts, got["stepAttempts"])
		})
	}
}
