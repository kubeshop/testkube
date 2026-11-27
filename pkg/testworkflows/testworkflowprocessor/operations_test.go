package testworkflowprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestStubAgent(t *testing.T) {
	tests := []struct {
		name    string
		step    testworkflowsv1.Step
		wantErr error
	}{
		{
			name: "ignores a step with no agent operation",
			step: testworkflowsv1.Step{StepOperations: testworkflowsv1.StepOperations{Shell: "true"}},
		},
		{
			name: "refuses a step with an agent operation",
			step: testworkflowsv1.Step{StepOperations: testworkflowsv1.StepOperations{
				Agent: &testworkflowsv1.StepAgent{Name: "failure-triage"},
			}},
			wantErr: constants.ErrOpenSourceAgentOperationIsNotAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stage, err := StubAgent(nil, nil, nil, tt.step)

			assert.Nil(t, stage)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
