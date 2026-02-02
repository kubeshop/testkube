/*
 * Testkube API
 *
 * Tests for YAML unmarshaling of boxed types.
 */
package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBoxedString_UnmarshalYAML(t *testing.T) {
	t.Run("shorthand string form", func(t *testing.T) {
		input := `value: hello world`
		var result struct {
			Value *BoxedString `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, "hello world", result.Value.Value)
	})

	t.Run("object form with value field", func(t *testing.T) {
		input := `value:
  value: hello world`
		var result struct {
			Value *BoxedString `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, "hello world", result.Value.Value)
	})

	t.Run("empty string", func(t *testing.T) {
		input := `value: ""`
		var result struct {
			Value *BoxedString `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, "", result.Value.Value)
	})
}

func TestBoxedBoolean_UnmarshalYAML(t *testing.T) {
	t.Run("shorthand true form", func(t *testing.T) {
		input := `value: true`
		var result struct {
			Value *BoxedBoolean `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.True(t, result.Value.Value)
	})

	t.Run("shorthand false form", func(t *testing.T) {
		input := `value: false`
		var result struct {
			Value *BoxedBoolean `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.False(t, result.Value.Value)
	})

	t.Run("object form with value field", func(t *testing.T) {
		input := `value:
  value: true`
		var result struct {
			Value *BoxedBoolean `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.True(t, result.Value.Value)
	})
}

func TestBoxedInteger_UnmarshalYAML(t *testing.T) {
	t.Run("shorthand integer form", func(t *testing.T) {
		input := `value: 42`
		var result struct {
			Value *BoxedInteger `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, int32(42), result.Value.Value)
	})

	t.Run("object form with value field", func(t *testing.T) {
		input := `value:
  value: 42`
		var result struct {
			Value *BoxedInteger `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, int32(42), result.Value.Value)
	})

	t.Run("zero value", func(t *testing.T) {
		input := `value: 0`
		var result struct {
			Value *BoxedInteger `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, int32(0), result.Value.Value)
	})
}

func TestBoxedStringList_UnmarshalYAML(t *testing.T) {
	t.Run("shorthand array form", func(t *testing.T) {
		input := `value:
  - one
  - two
  - three`
		var result struct {
			Value *BoxedStringList `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, []string{"one", "two", "three"}, result.Value.Value)
	})

	t.Run("object form with value field", func(t *testing.T) {
		input := `value:
  value:
    - one
    - two
    - three`
		var result struct {
			Value *BoxedStringList `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, []string{"one", "two", "three"}, result.Value.Value)
	})

	t.Run("empty array", func(t *testing.T) {
		input := `value: []`
		var result struct {
			Value *BoxedStringList `yaml:"value"`
		}
		err := yaml.Unmarshal([]byte(input), &result)
		require.NoError(t, err)
		require.NotNil(t, result.Value)
		assert.Equal(t, []string{}, result.Value.Value)
	})
}

// TestWorkflowWithBoxedTypes verifies that a full workflow with boxed types
// can be unmarshaled from YAML.
func TestWorkflowWithBoxedTypes(t *testing.T) {
	t.Run("service with shorthand shell", func(t *testing.T) {
		input := `
name: test-workflow
spec:
  services:
    myservice:
      image: alpine
      shell: "echo hello"
`
		var wf TestWorkflow
		err := yaml.Unmarshal([]byte(input), &wf)
		require.NoError(t, err)
		assert.Equal(t, "test-workflow", wf.Name)
		require.NotNil(t, wf.Spec)
		require.Contains(t, wf.Spec.Services, "myservice")
		require.NotNil(t, wf.Spec.Services["myservice"].Shell)
		assert.Equal(t, "echo hello", wf.Spec.Services["myservice"].Shell.Value)
	})

	t.Run("service with object form shell", func(t *testing.T) {
		input := `
name: test-workflow
spec:
  services:
    myservice:
      image: alpine
      shell:
        value: "echo hello"
`
		var wf TestWorkflow
		err := yaml.Unmarshal([]byte(input), &wf)
		require.NoError(t, err)
		assert.Equal(t, "test-workflow", wf.Name)
		require.NotNil(t, wf.Spec)
		require.Contains(t, wf.Spec.Services, "myservice")
		require.NotNil(t, wf.Spec.Services["myservice"].Shell)
		assert.Equal(t, "echo hello", wf.Spec.Services["myservice"].Shell.Value)
	})

	t.Run("complex workflow with multiple boxed types", func(t *testing.T) {
		input := `
name: complex-workflow
spec:
  services:
    db:
      image: postgres
      shell: "pg_isready"
      command:
        - postgres
        - -c
        - max_connections=100
      args:
        value:
          - --config
          - /etc/config.yaml
`
		var wf TestWorkflow
		err := yaml.Unmarshal([]byte(input), &wf)
		require.NoError(t, err)
		assert.Equal(t, "complex-workflow", wf.Name)

		db := wf.Spec.Services["db"]
		require.NotNil(t, db.Shell)
		assert.Equal(t, "pg_isready", db.Shell.Value)

		require.NotNil(t, db.Command)
		assert.Equal(t, []string{"postgres", "-c", "max_connections=100"}, db.Command.Value)

		require.NotNil(t, db.Args)
		assert.Equal(t, []string{"--config", "/etc/config.yaml"}, db.Args.Value)
	})
}
