package presets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

func TestNewOpenSource_RefusesAgentStep(t *testing.T) {
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{{StepOperations: testworkflowsv1.StepOperations{
				Agent: &testworkflowsv1.StepAgent{Name: "failure-triage"},
			}}},
		},
	}

	_, err := NewOpenSource(ins).Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})

	assert.ErrorIs(t, err, constants.ErrOpenSourceAgentOperationIsNotAvailable)
}
