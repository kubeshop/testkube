package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestWorkflowStep_AgentSteps(t *testing.T) {
	tests := []struct {
		name string
		step TestWorkflowStep
		want []string
	}{
		{name: "no agent", step: TestWorkflowStep{Shell: "true"}},
		{
			name: "own agent, setup, steps, then parallel",
			step: TestWorkflowStep{
				Agent:    agentNamed("own"),
				Setup:    []TestWorkflowStep{{Agent: agentNamed("setup")}},
				Steps:    []TestWorkflowStep{{Agent: agentNamed("steps")}},
				Parallel: &TestWorkflowStepParallel{Agent: agentNamed("parallel")},
			},
			want: []string{"own", "setup", "steps", "parallel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, agentNames(tt.step.AgentSteps()))
		})
	}
}
