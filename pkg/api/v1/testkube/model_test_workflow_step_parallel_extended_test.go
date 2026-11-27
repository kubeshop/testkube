package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestWorkflowStepParallel_AgentSteps(t *testing.T) {
	tests := []struct {
		name     string
		parallel TestWorkflowStepParallel
		want     []string
	}{
		{name: "no agent", parallel: TestWorkflowStepParallel{Shell: "true"}},
		{
			name: "own agent, setup, steps, then after",
			parallel: TestWorkflowStepParallel{
				Agent: agentNamed("own"),
				Setup: []TestWorkflowStep{{Agent: agentNamed("setup")}},
				Steps: []TestWorkflowStep{{Agent: agentNamed("steps")}},
				After: []TestWorkflowStep{{Agent: agentNamed("after")}},
			},
			want: []string{"own", "setup", "steps", "after"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, agentNames(tt.parallel.AgentSteps()))
		})
	}
}
