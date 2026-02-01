package formatters

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatListAgents(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatListAgents, `{"elements":[]}`)

	t.Run("parses JSON input with full agent data", func(t *testing.T) {
		input := `{
			"elements": [{
				"id": "agent-123",
				"name": "production-runner",
				"version": "1.2.3",
				"namespace": "testkube",
				"disabled": false,
				"floating": true,
				"labels": {"env": "prod", "team": "platform"},
				"environments": [{"id": "env-1", "name": "prod", "slug": "prod"}],
				"capabilities": ["runner", "listener"],
				"accessedAt": "2025-01-20T15:00:00Z",
				"createdAt": "2025-01-01T10:00:00Z",
				"runnerPolicy": {"requiredMatch": ["label=value"]},
				"isSuperAgent": true
			}]
		}`

		result, err := FormatListAgents(input)
		require.NoError(t, err)

		var output formattedAgentsResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.Len(t, output.Elements, 1)
		agent := output.Elements[0]
		assert.Equal(t, "agent-123", agent.ID)
		assert.Equal(t, "production-runner", agent.Name)
		assert.Equal(t, "1.2.3", agent.Version)
		assert.Equal(t, "testkube", agent.Namespace)
		assert.False(t, agent.Disabled)
		assert.Equal(t, []string{"runner", "listener"}, agent.Capabilities)
		assert.True(t, agent.IsSuperAgent)

		// Verify stripped fields are not in output
		assert.NotContains(t, result, "environments")
		assert.NotContains(t, result, "accessedAt")
		assert.NotContains(t, result, "createdAt")
		assert.NotContains(t, result, "labels")
		assert.NotContains(t, result, "runnerPolicy")
		assert.NotContains(t, result, "floating")
	})

	t.Run("parses JSON input with minimal agent data", func(t *testing.T) {
		input := `{
			"elements": [{"id": "agent-minimal", "name": "minimal"}]
		}`

		result, err := FormatListAgents(input)
		require.NoError(t, err)

		var output formattedAgentsResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.Len(t, output.Elements, 1)
		agent := output.Elements[0]
		assert.Equal(t, "agent-minimal", agent.ID)
		assert.Equal(t, "minimal", agent.Name)
		assert.Empty(t, agent.Version)
		assert.Empty(t, agent.Namespace)
		assert.False(t, agent.Disabled)
		assert.Nil(t, agent.Capabilities)
		assert.False(t, agent.IsSuperAgent)
	})

	t.Run("handles empty elements array", func(t *testing.T) {
		input := `{"elements": []}`

		result, err := FormatListAgents(input)
		require.NoError(t, err)

		var output formattedAgentsResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Empty(t, output.Elements)
	})

	t.Run("handles multiple agents", func(t *testing.T) {
		input := `{
			"elements": [
				{"id": "agent-1", "name": "runner-1", "capabilities": ["runner"]},
				{"id": "agent-2", "name": "runner-2", "disabled": true},
				{"id": "agent-3", "name": "super-agent", "isSuperAgent": true}
			]
		}`

		result, err := FormatListAgents(input)
		require.NoError(t, err)

		var output formattedAgentsResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.Len(t, output.Elements, 3)
		assert.Equal(t, "runner-1", output.Elements[0].Name)
		assert.Equal(t, []string{"runner"}, output.Elements[0].Capabilities)
		assert.Equal(t, "runner-2", output.Elements[1].Name)
		assert.True(t, output.Elements[1].Disabled)
		assert.Equal(t, "super-agent", output.Elements[2].Name)
		assert.True(t, output.Elements[2].IsSuperAgent)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatListAgents(input)
		assert.Error(t, err)
	})
}
