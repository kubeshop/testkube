package testkube

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
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

func TestTemplatableBoxedInteger_MarshalJSON(t *testing.T) {
	t.Run("integer value remains numeric", func(t *testing.T) {
		value := TemplatableBoxedInteger{Value: "65532"}

		raw, err := json.Marshal(value)

		require.NoError(t, err)
		assert.Equal(t, `{"value":65532}`, string(raw))
	})

	t.Run("large integer value remains numeric", func(t *testing.T) {
		value := TemplatableBoxedInteger{Value: "9223372036854775807"}

		raw, err := json.Marshal(value)

		require.NoError(t, err)
		assert.Equal(t, `{"value":9223372036854775807}`, string(raw))
	})

	t.Run("template value remains string", func(t *testing.T) {
		value := TemplatableBoxedInteger{Value: "{{ config.runAsUser }}"}

		raw, err := json.Marshal(value)

		require.NoError(t, err)
		assert.JSONEq(t, `{"value":"{{ config.runAsUser }}"}`, string(raw))
	})
}

func TestTemplatableBoxedInteger_UnmarshalBSON(t *testing.T) {
	type oldBoxedInteger struct {
		Value int32 `bson:"value"`
	}

	type oldSecurityContext struct {
		RunAsUser *oldBoxedInteger `bson:"runAsUser"`
	}

	type newSecurityContext struct {
		RunAsUser *TemplatableBoxedInteger `bson:"runAsUser"`
	}

	t.Run("legacy embedded document with integer value", func(t *testing.T) {
		raw, err := bson.Marshal(oldSecurityContext{
			RunAsUser: &oldBoxedInteger{Value: 65532},
		})
		require.NoError(t, err)

		var decoded newSecurityContext
		err = bson.Unmarshal(raw, &decoded)
		require.NoError(t, err)
		require.NotNil(t, decoded.RunAsUser)
		assert.Equal(t, "65532", decoded.RunAsUser.Value)
	})

	t.Run("new embedded document with string value", func(t *testing.T) {
		raw, err := bson.Marshal(newSecurityContext{
			RunAsUser: &TemplatableBoxedInteger{Value: "{{ config.runAsUser }}"},
		})
		require.NoError(t, err)

		var decoded newSecurityContext
		err = bson.Unmarshal(raw, &decoded)
		require.NoError(t, err)
		require.NotNil(t, decoded.RunAsUser)
		assert.Equal(t, "{{ config.runAsUser }}", decoded.RunAsUser.Value)
	})
}
