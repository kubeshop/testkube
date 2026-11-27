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
			name: "fills the messages, the reason codes, and the step attempts keyed by step id",
			execution: &testkube.TestWorkflowExecution{
				Id:        "exec-1",
				Name:      "wf-1",
				Workflow:  &testkube.TestWorkflow{Name: "wf"},
				Signature: []testkube.TestWorkflowSignature{{Ref: "rstep1", Id: "run_test"}, {Ref: "rstep2", Id: "report"}},
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
				StepErrors:   map[string]string{"run_test": "the step timed out"},
				StepAttempts: map[string]int64{"run_test": 3},
				ErrorReason:  "unschedulable",
				StepReasons:  map[string]string{"run_test": "step-timeout"},
			},
		},
		{
			name: "keys a step by its ref when it has no id, repeats the id of an unrelated step, or has no signature",
			execution: &testkube.TestWorkflowExecution{
				Signature: []testkube.TestWorkflowSignature{
					{Ref: "rgroup", Id: "build", Children: []testkube.TestWorkflowSignature{
						{Ref: "rclone", Category: "Clone Git repository"},
						{Ref: "rbare"},
					}},
					{Ref: "rrepeat", Id: "build", Children: []testkube.TestWorkflowSignature{
						{Ref: "rrepeatshell", Id: "build"},
					}},
				},
				Result: &testkube.TestWorkflowResult{
					Steps: map[string]testkube.TestWorkflowStepResult{
						"rgroup":       {ErrorMessage: "a child step failed"},
						"rclone":       {ErrorMessage: "the repository does not exist"},
						"rbare":        {ErrorMessage: "the step has no name"},
						"rrepeat":      {ErrorMessage: "another step failed"},
						"rrepeatshell": {ErrorMessage: "its command failed"},
						"rnosignature": {ErrorMessage: "the signature does not hold the step"},
					},
				},
			},
			want: Execution{
				Outputs: map[string]string{},
				StepErrors: map[string]string{
					"build":        "a child step failed",
					"rclone":       "the repository does not exist",
					"rbare":        "the step has no name",
					"rrepeat":      "another step failed",
					"rrepeatshell": "its command failed",
					"rnosignature": "the signature does not hold the step",
				},
			},
		},
		{
			name: "keys a group and the stage that runs its operation by the id of the step, and the stage wins",
			execution: &testkube.TestWorkflowExecution{
				Signature: []testkube.TestWorkflowSignature{
					{Ref: "rgroup", Id: "build", Name: "Build", Children: []testkube.TestWorkflowSignature{
						{Ref: "rshell", Id: "build"},
						{Ref: "rartifacts", Id: "build_upload_artifacts", Category: "Upload artifacts"},
					}},
				},
				Result: &testkube.TestWorkflowResult{
					Steps: map[string]testkube.TestWorkflowStepResult{
						"rgroup": {ErrorMessage: "a child step failed", Attempts: 1},
						"rshell": {ErrorMessage: "the command failed", Attempts: 2},
					},
				},
			},
			want: Execution{
				Outputs:      map[string]string{},
				StepErrors:   map[string]string{"build": "the command failed"},
				StepAttempts: map[string]int64{"build": 2},
			},
		},
		{
			name: "keys the value of a group by the id of the step when the stage that runs its operation has none",
			execution: &testkube.TestWorkflowExecution{
				Signature: []testkube.TestWorkflowSignature{
					{Ref: "rgroup", Id: "build", Name: "Build", Children: []testkube.TestWorkflowSignature{
						{Ref: "rshell", Id: "build"},
					}},
				},
				Result: &testkube.TestWorkflowResult{
					Steps: map[string]testkube.TestWorkflowStepResult{
						"rgroup": {ErrorMessage: "the step timed out", ErrorReason: "step-timeout"},
						"rshell": {},
					},
					StatusDetails: &testkube.TestWorkflowStatusDetails{
						Type_:  string(testkube.StatusDetailsTypeStepFailure),
						Reason: "step-timeout",
						Step:   "rgroup",
					},
				},
			},
			want: Execution{
				Outputs:      map[string]string{},
				StepErrors:   map[string]string{"build": "the step timed out"},
				StepReasons:  map[string]string{"build": "step-timeout"},
				StatusType:   string(testkube.StatusDetailsTypeStepFailure),
				StatusReason: "step-timeout",
				StatusStep:   "build",
			},
		},
		{
			name: "fills the status fields from the object, keys the step by id, and leaves the message out",
			execution: &testkube.TestWorkflowExecution{
				Id:        "exec-2",
				Name:      "wf-2",
				Signature: []testkube.TestWorkflowSignature{{Ref: "rstep1", Id: "run_test"}},
				Result: &testkube.TestWorkflowResult{
					Status:         common.Ptr(testkube.ABORTED_TestWorkflowStatus),
					Initialization: &testkube.TestWorkflowStepResult{},
					StatusDetails: &testkube.TestWorkflowStatusDetails{
						Type_:   string(testkube.StatusDetailsTypeExecutionFailure),
						Reason:  string(testkube.StopReasonOOMKilled),
						Step:    "rstep1",
						Message: "the container exceeded its memory limit",
						Actor:   string(testkube.StopActorRunner),
						User:    &testkube.TestWorkflowStatusDetailsUser{Name: "Ada", Email: "ada@example.com"},
					},
				},
			},
			want: Execution{
				Id:           "exec-2",
				Name:         "wf-2",
				Status:       "aborted",
				Outputs:      map[string]string{},
				StatusType:   string(testkube.StatusDetailsTypeExecutionFailure),
				StatusReason: string(testkube.StopReasonOOMKilled),
				StatusStep:   "run_test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FromExecution(tt.execution))
		})
	}
}

func TestExecution_SetStatusDetails(t *testing.T) {
	tests := []struct {
		name       string
		execution  *testkube.TestWorkflowExecution
		wantType   string
		wantReason string
		wantStep   string
	}{
		{
			name: "copies the codes and leaves the message and the user out",
			execution: &testkube.TestWorkflowExecution{Result: &testkube.TestWorkflowResult{
				StatusDetails: &testkube.TestWorkflowStatusDetails{
					Type_:   string(testkube.StatusDetailsTypeInitFailure),
					Reason:  string(testkube.StopReasonUnschedulable),
					Step:    "rstep1",
					Message: "no node can run the pod",
					User:    &testkube.TestWorkflowStatusDetailsUser{Name: "Ada", Email: "ada@example.com"},
				},
			}},
			wantType:   string(testkube.StatusDetailsTypeInitFailure),
			wantReason: string(testkube.StopReasonUnschedulable),
			wantStep:   "rstep1",
		},
		{name: "leaves the codes empty without the object", execution: &testkube.TestWorkflowExecution{Result: &testkube.TestWorkflowResult{}}},
		{name: "leaves the codes empty without a result", execution: &testkube.TestWorkflowExecution{}},
		{name: "leaves the codes empty without an execution"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := Execution{}

			entry.SetStatusDetails(tt.execution)

			assert.Equal(t, tt.wantType, entry.StatusType)
			assert.Equal(t, tt.wantReason, entry.StatusReason)
			assert.Equal(t, tt.wantStep, entry.StatusStep)
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
