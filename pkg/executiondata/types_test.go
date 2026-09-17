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
			name: "fills the messages, the reason codes, and the step attempts keyed by ref",
			execution: &testkube.TestWorkflowExecution{
				Id:       "exec-1",
				Name:     "wf-1",
				Workflow: &testkube.TestWorkflow{Name: "wf"},
				Result: &testkube.TestWorkflowResult{
					Status:         common.Ptr(testkube.ABORTED_TestWorkflowStatus),
					Initialization: &testkube.TestWorkflowStepResult{ErrorMessage: "the pod cannot be scheduled", ErrorReason: "unschedulable"},
					Steps: map[string]testkube.TestWorkflowStepResult{
						"rstep1": {ErrorMessage: "the step timed out", ErrorReason: "step-timeout", Attempts: 3},
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
				ErrorReason:  "unschedulable",
				StepReasons:  map[string]string{"rstep1": "step-timeout"},
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
		name      string
		execution Execution
		want      map[string]interface{}
	}{
		{
			name: "exposes the messages, the reason codes, and the step attempts",
			execution: Execution{
				ErrorMessage: "the pod cannot be scheduled",
				StepErrors:   map[string]string{"rstep1": "the step timed out"},
				StepAttempts: map[string]int64{"rstep1": 3},
				ErrorReason:  "unschedulable",
				StepReasons:  map[string]string{"rstep1": "step-timeout"},
			},
			want: map[string]interface{}{
				"errorMessage": "the pod cannot be scheduled",
				"stepErrors":   map[string]interface{}{"rstep1": "the step timed out"},
				"stepAttempts": map[string]interface{}{"rstep1": int64(3)},
				"errorReason":  "unschedulable",
				"stepReasons":  map[string]interface{}{"rstep1": "step-timeout"},
			},
		},
		{
			name:      "exposes empty values when the execution has no errors and no attempts",
			execution: Execution{},
			want: map[string]interface{}{
				"errorMessage": "",
				"stepErrors":   map[string]interface{}{},
				"stepAttempts": map[string]interface{}{},
				"errorReason":  "",
				"stepReasons":  map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.execution.AsMap()

			for key, want := range tt.want {
				assert.Equal(t, want, got[key], key)
			}
		})
	}
}
