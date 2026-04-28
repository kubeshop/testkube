package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	sigsyaml "sigs.k8s.io/yaml"
)

func TestWorkflowInt64OrString_UnmarshalYAML(t *testing.T) {
	t.Run("numeric value above int32 range", func(t *testing.T) {
		var value struct {
			RunAsUser *WorkflowInt64OrString `yaml:"runAsUser"`
		}

		err := yaml.Unmarshal([]byte("runAsUser: 4294967295"), &value)
		require.NoError(t, err)
		require.NotNil(t, value.RunAsUser)

		resolved, err := ResolveWorkflowInt64("runAsUser", value.RunAsUser)
		require.NoError(t, err)
		assert.Equal(t, int64(4294967295), *resolved)
	})

	t.Run("template string", func(t *testing.T) {
		var value struct {
			RunAsUser *WorkflowInt64OrString `yaml:"runAsUser"`
		}

		err := yaml.Unmarshal([]byte("runAsUser: '{{ config.runAsUser }}'"), &value)
		require.NoError(t, err)
		require.NotNil(t, value.RunAsUser)
		assert.Equal(t, "{{ config.runAsUser }}", value.RunAsUser.String())
	})
}

func TestWorkflowInt64OrString_UnmarshalJSON(t *testing.T) {
	var value struct {
		RunAsUser *WorkflowInt64OrString `json:"runAsUser"`
	}

	err := json.Unmarshal([]byte(`{"runAsUser":4294967295}`), &value)
	require.NoError(t, err)
	require.NotNil(t, value.RunAsUser)

	resolved, err := ResolveWorkflowInt64("runAsUser", value.RunAsUser)
	require.NoError(t, err)
	assert.Equal(t, int64(4294967295), *resolved)
}

func TestWorkflowSecurityContextToKubePreservesHighNumericIDs(t *testing.T) {
	input := []byte(`
container:
  runAsUser: 4294967295
  runAsGroup: 4294967294
pod:
  runAsUser: 4294967293
  runAsGroup: 4294967292
  fsGroup: 4294967291
`)

	var value struct {
		Container WorkflowSecurityContext    `json:"container" yaml:"container"`
		Pod       WorkflowPodSecurityContext `json:"pod" yaml:"pod"`
	}
	err := sigsyaml.Unmarshal(input, &value)
	require.NoError(t, err)

	container, err := value.Container.ToKube()
	require.NoError(t, err)
	require.NotNil(t, container)
	assert.Equal(t, int64(4294967295), *container.RunAsUser)
	assert.Equal(t, int64(4294967294), *container.RunAsGroup)

	pod, err := value.Pod.ToKube()
	require.NoError(t, err)
	require.NotNil(t, pod)
	assert.Equal(t, int64(4294967293), *pod.RunAsUser)
	assert.Equal(t, int64(4294967292), *pod.RunAsGroup)
	assert.Equal(t, int64(4294967291), *pod.FSGroup)
}
