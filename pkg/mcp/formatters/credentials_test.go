package formatters

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseFormattedCredentials(t *testing.T, raw string) formattedCredentialsResult {
	t.Helper()
	var out formattedCredentialsResult
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	return out
}

func TestFormatListCredentials(t *testing.T) {
	// Empty / null / whitespace input → empty credentials list.
	RunEmptyInputCases(t, FormatListCredentials, `{"credentials":[]}`)

	t.Run("happy path: org and environment scoped credentials", func(t *testing.T) {
		input := `{"elements":[
			{"name":"github-access-token","type":"secret","reference":"github-access-token"},
			{"name":"staging-api-key","type":"secret","reference":"staging-api-key","environmentId":"tkcenv_1"}
		]}`

		result, err := FormatListCredentials(input)
		require.NoError(t, err)
		out := parseFormattedCredentials(t, result)
		require.Len(t, out.Credentials, 2)

		assert.Equal(t, "github-access-token", out.Credentials[0].Name)
		assert.Equal(t, "secret", out.Credentials[0].Type)
		assert.Equal(t, "organization", out.Credentials[0].Scope)
		assert.Equal(t, `credential("github-access-token")`, out.Credentials[0].Expression)

		// env-scoped credential is annotated as "environment".
		assert.Equal(t, "environment", out.Credentials[1].Scope)
	})

	t.Run("never returns secret values", func(t *testing.T) {
		input := `{"elements":[
			{"name":"db-password","type":"secret","reference":"db-password","value":"super-secret-pw","base64Value":"c3VwZXItc2VjcmV0LXB3"}
		]}`

		result, err := FormatListCredentials(input)
		require.NoError(t, err)
		assert.NotContains(t, result, "super-secret-pw")
		assert.NotContains(t, result, "c3VwZXItc2VjcmV0LXB3")
		// Reference/name still surfaced so the agent can use it.
		assert.Contains(t, result, "db-password")
	})

	t.Run("drops variable values too", func(t *testing.T) {
		// `variable` is not a secret, but the tool's contract is "never values".
		input := `{"elements":[
			{"name":"api-endpoint","type":"variable","reference":"api-endpoint","value":"https://staging.example.com"}
		]}`

		result, err := FormatListCredentials(input)
		require.NoError(t, err)
		assert.NotContains(t, result, "https://staging.example.com")
		out := parseFormattedCredentials(t, result)
		require.Len(t, out.Credentials, 1)
		assert.Equal(t, "variable", out.Credentials[0].Type)
	})

	t.Run("omits non-referenceable execution-scoped credentials", func(t *testing.T) {
		// Execution-scoped credentials come back with an empty reference.
		input := `{"elements":[
			{"name":"exec-temp-token","type":"secret","reference":"","executionId":"6a34c733"},
			{"name":"real-token","type":"secret","reference":"real-token"}
		]}`

		result, err := FormatListCredentials(input)
		require.NoError(t, err)
		out := parseFormattedCredentials(t, result)
		require.Len(t, out.Credentials, 1)
		assert.Equal(t, "real-token", out.Credentials[0].Name)
	})

	t.Run("renders ready-to-use credential expression", func(t *testing.T) {
		input := `{"elements":[{"name":"gh","type":"secret","reference":"github-access-token"}]}`
		result, err := FormatListCredentials(input)
		require.NoError(t, err)
		assert.True(t, strings.Contains(result, `credential(\"github-access-token\")`),
			"expected escaped credential expression in JSON output, got: %s", result)
	})
}
