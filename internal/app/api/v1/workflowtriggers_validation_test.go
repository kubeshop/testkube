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

func TestValidateWorkflowTriggerSpec_InvalidSpec_MissingEvent(t *testing.T) {
	t.Parallel()

	trigger := &testkube.WorkflowTrigger{
		Watch: &testkube.WorkflowTriggerWatch{
			Resource: testkube.WorkflowTriggerResource{Kind: "Deployment"},
		},
		Run: testkube.WorkflowTriggerRun{
			Workflow: testkube.WorkflowTriggerWorkflowSelector{Name: "workflow"},
		},
	}

	err := validateWorkflowTriggerSpec(trigger)
	require.Error(t, err)
	require.ErrorContains(t, err, "when.event is required")
}

func TestValidateWorkflowTriggerSpec_InvalidSpec_MissingWorkflowSelector(t *testing.T) {
	t.Parallel()

	trigger := &testkube.WorkflowTrigger{
		Watch: &testkube.WorkflowTriggerWatch{
			Resource: testkube.WorkflowTriggerResource{Kind: "Deployment"},
		},
		When: testkube.WorkflowTriggerWhen{Event: "modified"},
		Run:  testkube.WorkflowTriggerRun{},
	}

	err := validateWorkflowTriggerSpec(trigger)
	require.Error(t, err)
	require.ErrorContains(t, err, "run.workflow must specify")
}
