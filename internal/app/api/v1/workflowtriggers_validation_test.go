package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestValidateWorkflowTriggerSpec_Valid(t *testing.T) {
	t.Parallel()

	trigger := &testkube.WorkflowTrigger{
		Watch: &testkube.WorkflowTriggerWatch{
			Resource: testkube.WorkflowTriggerResource{Kind: "Deployment"},
		},
		When: testkube.WorkflowTriggerWhen{Event: "modified"},
		Run: testkube.WorkflowTriggerRun{
			Workflow: testkube.WorkflowTriggerWorkflowSelector{Name: "workflow"},
		},
	}

	require.NoError(t, validateWorkflowTriggerSpec(trigger))
}

func TestValidateWorkflowTriggerSpec_InvalidDelay(t *testing.T) {
	t.Parallel()

	trigger := &testkube.WorkflowTrigger{
		Watch: &testkube.WorkflowTriggerWatch{
			Resource: testkube.WorkflowTriggerResource{Kind: "Deployment"},
		},
		When: testkube.WorkflowTriggerWhen{Event: "modified"},
		Run: testkube.WorkflowTriggerRun{
			Delay:    "not-a-duration",
			Workflow: testkube.WorkflowTriggerWorkflowSelector{Name: "workflow"},
		},
	}

	require.Error(t, validateWorkflowTriggerSpec(trigger))
}
