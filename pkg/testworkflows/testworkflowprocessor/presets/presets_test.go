package presets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestPresets_RefuseAgentStep(t *testing.T) {
	tests := []struct {
		name      string
		processor testworkflowprocessor.Processor
		wantErr   error
	}{
		{name: "the open source preset names the standalone mode", processor: NewOpenSource(ins), wantErr: constants.ErrOpenSourceAgentOperationIsNotAvailable},
		{name: "the commercial preset names the Runner version", processor: NewPro(ins), wantErr: errAgentOperationHasNoProcessor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &testworkflowsv1.TestWorkflow{
				Spec: testworkflowsv1.TestWorkflowSpec{
					Steps: []testworkflowsv1.Step{{StepOperations: testworkflowsv1.StepOperations{
						Agent: &testworkflowsv1.StepAgent{Name: "failure-triage"},
					}}},
				},
			}

			_, err := tt.processor.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
