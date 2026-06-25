package agents

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/cloud/client"
)

func TestBuildUpdateAgentInputDeleteLabelWithoutRunnerPolicy(t *testing.T) {
	agent := &client.Agent{
		Labels: map[string]string{
			"testkube.io/source": "helm",
			"team":               "qa",
		},
	}

	input, err := buildUpdateAgentInput(agent, nil, []string{"testkube.io/source"}, "", "")

	require.NoError(t, err)
	require.NotNil(t, input.Labels)
	require.Equal(t, map[string]string{"team": "qa"}, *input.Labels)
	require.Nil(t, input.RunnerPolicy)
}

func TestBuildUpdateAgentInputDeleteLabelWithEmptyRunnerPolicy(t *testing.T) {
	agent := &client.Agent{
		Labels:       map[string]string{"testkube.io/source": "helm"},
		RunnerPolicy: &client.RunnerPolicy{},
	}

	input, err := buildUpdateAgentInput(agent, nil, []string{"testkube.io/source"}, "", "")

	require.NoError(t, err)
	require.NotNil(t, input.Labels)
	require.Empty(t, *input.Labels)
	require.NotNil(t, input.RunnerPolicy)
	require.Empty(t, input.RunnerPolicy.RequiredMatch)
}

func TestBuildUpdateAgentInputDoesNotDeleteGroupLabelFromGroupRunner(t *testing.T) {
	agent := &client.Agent{
		Labels:       map[string]string{"group": "payments"},
		RunnerPolicy: &client.RunnerPolicy{RequiredMatch: []string{"group"}},
	}

	_, err := buildUpdateAgentInput(agent, nil, []string{"group"}, "", "")

	require.EqualError(t, err, "cannot delete group label when runner mode is group")
}

func TestBuildUpdateAgentInputSetsGroupName(t *testing.T) {
	agent := &client.Agent{}

	input, err := buildUpdateAgentInput(agent, nil, nil, "group", "payments")

	require.NoError(t, err)
	require.NotNil(t, input.Labels)
	require.Equal(t, "payments", (*input.Labels)["group"])
	require.NotNil(t, input.RunnerPolicy)
	require.Equal(t, []string{"group"}, input.RunnerPolicy.RequiredMatch)
}
