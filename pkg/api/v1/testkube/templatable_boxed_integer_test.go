package testkube

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatableBoxedInteger_UnmarshalJSON(t *testing.T) {
	t.Run("scalar integer", func(t *testing.T) {
		var value TemplatableBoxedInteger
		err := json.Unmarshal([]byte("65532"), &value)
		require.NoError(t, err)
		assert.Equal(t, "65532", value.Value)
	})

	t.Run("scalar template string", func(t *testing.T) {
		var value TemplatableBoxedInteger
		err := json.Unmarshal([]byte(`"{{ config.runAsUser }}"`), &value)
		require.NoError(t, err)
		assert.Equal(t, "{{ config.runAsUser }}", value.Value)
	})

	t.Run("object integer value", func(t *testing.T) {
		var value TemplatableBoxedInteger
		err := json.Unmarshal([]byte(`{"value":65532}`), &value)
		require.NoError(t, err)
		assert.Equal(t, "65532", value.Value)
	})

	t.Run("object template value", func(t *testing.T) {
		var value TemplatableBoxedInteger
		err := json.Unmarshal([]byte(`{"value":"{{ config.runAsUser }}"}`), &value)
		require.NoError(t, err)
		assert.Equal(t, "{{ config.runAsUser }}", value.Value)
	})
}
