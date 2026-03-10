package formatters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEmptyInput(t *testing.T) {
	t.Run("empty string returns true", func(t *testing.T) {
		assert.True(t, IsEmptyInput(""))
	})

	t.Run("whitespace string returns true", func(t *testing.T) {
		assert.True(t, IsEmptyInput("   \n\t  "))
	})

	t.Run("null string returns true", func(t *testing.T) {
		assert.True(t, IsEmptyInput("null"))
	})

	t.Run("null with whitespace returns true", func(t *testing.T) {
		assert.True(t, IsEmptyInput("  null  "))
	})

	t.Run("valid JSON returns false", func(t *testing.T) {
		assert.False(t, IsEmptyInput("[]"))
		assert.False(t, IsEmptyInput("{}"))
		assert.False(t, IsEmptyInput(`{"key": "value"}`))
	})
}

func TestParseJSON(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	t.Run("parses JSON object", func(t *testing.T) {
		input := `{"name": "test", "value": 42}`
		result, isEmpty, err := ParseJSON[testStruct](input)
		require.NoError(t, err)
		assert.False(t, isEmpty)
		assert.Equal(t, "test", result.Name)
		assert.Equal(t, 42, result.Value)
	})

	t.Run("parses JSON array", func(t *testing.T) {
		input := `[{"name": "a", "value": 1}, {"name": "b", "value": 2}]`
		result, isEmpty, err := ParseJSON[[]testStruct](input)
		require.NoError(t, err)
		assert.False(t, isEmpty)
		require.Len(t, result, 2)
		assert.Equal(t, "a", result[0].Name)
		assert.Equal(t, "b", result[1].Name)
	})

	t.Run("empty string returns isEmpty true", func(t *testing.T) {
		result, isEmpty, err := ParseJSON[testStruct]("")
		require.NoError(t, err)
		assert.True(t, isEmpty)
		assert.Equal(t, testStruct{}, result)
	})

	t.Run("null string returns isEmpty true", func(t *testing.T) {
		result, isEmpty, err := ParseJSON[testStruct]("null")
		require.NoError(t, err)
		assert.True(t, isEmpty)
		assert.Equal(t, testStruct{}, result)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, _, err := ParseJSON[testStruct]("{invalid}")
		require.Error(t, err)
	})

	t.Run("parses YAML input", func(t *testing.T) {
		input := "name: yaml-test\nvalue: 100"
		result, isEmpty, err := ParseJSON[testStruct](input)
		require.NoError(t, err)
		assert.False(t, isEmpty)
		assert.Equal(t, "yaml-test", result.Name)
		assert.Equal(t, 100, result.Value)
	})
}

func TestFormatJSON(t *testing.T) {
	t.Run("marshals struct to JSON", func(t *testing.T) {
		input := struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}{Name: "test", Value: 42}

		result, err := FormatJSON(input)
		require.NoError(t, err)
		assert.Equal(t, `{"name":"test","value":42}`, result)
	})

	t.Run("marshals array to JSON", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result, err := FormatJSON(input)
		require.NoError(t, err)
		assert.Equal(t, `["a","b","c"]`, result)
	})

	t.Run("marshals empty array to JSON", func(t *testing.T) {
		input := []string{}
		result, err := FormatJSON(input)
		require.NoError(t, err)
		assert.Equal(t, `[]`, result)
	})
}

// RunEmptyInputCases is a shared test helper that runs standard empty/null/whitespace tests
// for formatter functions. The emptyOutput parameter specifies what the formatter should
// return for empty inputs (e.g., "[]" for arrays, "{}" for objects).
func RunEmptyInputCases(t *testing.T, formatter func(string) (string, error), emptyOutput string) {
	t.Helper()

	t.Run("empty string returns expected output", func(t *testing.T) {
		result, err := formatter("")
		require.NoError(t, err)
		assert.Equal(t, emptyOutput, result)
	})

	t.Run("whitespace string returns expected output", func(t *testing.T) {
		result, err := formatter("   \n\t  ")
		require.NoError(t, err)
		assert.Equal(t, emptyOutput, result)
	})

	t.Run("null string returns expected output", func(t *testing.T) {
		result, err := formatter("null")
		require.NoError(t, err)
		assert.Equal(t, emptyOutput, result)
	})
}
