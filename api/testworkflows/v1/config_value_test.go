package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"
)

func TestConfigValue_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ConfigValue
	}{
		{name: "number above int32 range", input: `873760302410822`, want: "873760302410822"},
		{name: "number above int64 range", input: `18446744073709551615`, want: "18446744073709551615"},
		{name: "number in int32 range", input: `1024`, want: "1024"},
		{name: "negative number", input: `-5`, want: "-5"},
		{name: "float number", input: `1.5`, want: "1.5"},
		{name: "string", input: `"some value"`, want: "some value"},
		{name: "numeric string", input: `"873760302410822"`, want: "873760302410822"},
		{name: "template string", input: `"{{ config.id }}"`, want: "{{ config.id }}"},
		{name: "boolean", input: `true`, want: "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var value ConfigValue
			err := json.Unmarshal([]byte(tt.input), &value)
			require.NoError(t, err)
			assert.Equal(t, tt.want, value)
		})
	}

	t.Run("null keeps the value empty", func(t *testing.T) {
		var value struct {
			Default *ConfigValue `json:"default"`
		}
		err := json.Unmarshal([]byte(`{"default":null}`), &value)
		require.NoError(t, err)
		assert.Nil(t, value.Default)
	})

	t.Run("object is an error", func(t *testing.T) {
		var value ConfigValue
		err := json.Unmarshal([]byte(`{"a":1}`), &value)
		assert.Error(t, err)
	})
}

func TestConfigValue_MarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input ConfigValue
		want  string
	}{
		{name: "number in int32 range stays a number", input: "1024", want: `1024`},
		{name: "number above int32 range becomes a string", input: "873760302410822", want: `"873760302410822"`},
		{name: "number with a leading zero becomes a string", input: "007", want: `"007"`},
		{name: "plain string", input: "some value", want: `"some value"`},
		{name: "template string", input: "{{ config.id }}", want: `"{{ config.id }}"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestConfigValue_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ConfigValue
	}{
		{name: "number above int32 range", input: "default: 873760302410822", want: "873760302410822"},
		{name: "string", input: "default: 'some value'", want: "some value"},
		{name: "template string", input: "default: '{{ config.id }}'", want: "{{ config.id }}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var value struct {
				Default *ConfigValue `yaml:"default"`
			}
			err := yaml.Unmarshal([]byte(tt.input), &value)
			require.NoError(t, err)
			require.NotNil(t, value.Default)
			assert.Equal(t, tt.want, *value.Default)
		})
	}
}

func TestResources_UnmarshalLargeNumericMemory(t *testing.T) {
	containerJSON := []byte(`{
		"resources": {
			"limits": {"memory": 8589934592},
			"requests": {"cpu": "300m", "memory": 8589934592}
		}
	}`)

	var container ContainerConfig
	err := json.Unmarshal(containerJSON, &container)
	require.NoError(t, err)
	require.NotNil(t, container.Resources)
	assert.Equal(t, ConfigValue("8589934592"), container.Resources.Limits["memory"])
	assert.Equal(t, ConfigValue("300m"), container.Resources.Requests["cpu"])
	assert.Equal(t, ConfigValue("8589934592"), container.Resources.Requests["memory"])
}

func TestTestWorkflow_UnmarshalLargeNumericConfigDefault(t *testing.T) {
	workflowJSON := []byte(`{
		"apiVersion": "testworkflows.testkube.io/v1",
		"kind": "TestWorkflow",
		"metadata": {"name": "big-number"},
		"spec": {
			"config": {
				"id": {"type": "string", "default": 873760302410822}
			},
			"use": [{"name": "tpl", "config": {"id": 873760302410822}}]
		}
	}`)

	var workflow TestWorkflow
	err := json.Unmarshal(workflowJSON, &workflow)
	require.NoError(t, err)
	require.NotNil(t, workflow.Spec.Config["id"].Default)
	assert.Equal(t, "873760302410822", workflow.Spec.Config["id"].Default.String())
	assert.Equal(t, ConfigValue("873760302410822"), workflow.Spec.Use[0].Config["id"])

	workflowYAML := []byte(`
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: big-number
spec:
  config:
    id:
      type: string
      default: 873760302410822
`)
	var workflowFromYAML TestWorkflow
	err = sigsyaml.Unmarshal(workflowYAML, &workflowFromYAML)
	require.NoError(t, err)
	require.NotNil(t, workflowFromYAML.Spec.Config["id"].Default)
	assert.Equal(t, "873760302410822", workflowFromYAML.Spec.Config["id"].Default.String())
}
