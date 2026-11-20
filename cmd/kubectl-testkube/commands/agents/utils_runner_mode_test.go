package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"

	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

// TestGetAgentRunnerMode guards how the runner list describes a policy. A runner is
// reachable by name whatever its policy says, so the policy's other labels are the
// additional routes: a grouped runner that also lists "name" is the same runner as one
// that lists only "group", and calling it a custom policy tells the reader nothing.
func TestGetAgentRunnerMode(t *testing.T) {
	t.Parallel()

	agent := func(match ...string) *cloudclient.Agent {
		return &cloudclient.Agent{RunnerPolicy: &cloudclient.RunnerPolicy{RequiredMatch: match}}
	}

	cases := []struct {
		name  string
		agent *cloudclient.Agent
		want  string
		rule  string
	}{
		{"self-registered grouped runner", agent("group", "name"), "Group", "the shape Helm registration produces must not read as Custom Policy"},
		{"grouped runner", agent("group"), "Group", "the same runner, described the same way"},
		{"independent runner", agent("name"), "Independent", "unchanged"},
		{"global sentinel", agent("global"), "Global", "unchanged"},
		{"empty policy", agent(), "Global", "an empty policy is no policy"},
		{"no policy at all", &cloudclient.Agent{}, "Global", "unchanged"},
		{"nil agent", nil, "Global", "must not panic"},
		{"several labels", agent("region", "tier"), "Custom Policy", "the detail view lists the labels; the column stays one word"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, getAgentRunnerMode(tc.agent), tc.rule)
		})
	}
}
