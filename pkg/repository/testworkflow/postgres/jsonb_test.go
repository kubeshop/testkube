package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// A nil *T handed to an interface{} parameter is not a nil interface - it
// carries a type - so the obvious `v == nil` check misses it and the value is
// stored as the JSON literal "null" instead of SQL NULL.
func TestToJSONBStoresNothingForAnAbsentValue(t *testing.T) {
	t.Run("a typed nil pointer", func(t *testing.T) {
		data, err := toJSONB((*testkube.TestWorkflowRerun)(nil))
		require.NoError(t, err)
		assert.Nil(t, data, "a nil pointer is an absent value, not the four bytes \"null\"")
	})

	t.Run("a nil interface", func(t *testing.T) {
		data, err := toJSONB(nil)
		require.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("a nil map", func(t *testing.T) {
		data, err := toJSONB(map[string]string(nil))
		require.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("a nil slice", func(t *testing.T) {
		data, err := toJSONB([]string(nil))
		require.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("a value is still encoded", func(t *testing.T) {
		data, err := toJSONB(&testkube.TestWorkflowRerun{ExecutionId: "exec-1"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"executionId":"exec-1"}`, string(data))
	})

	// An empty struct is present, not absent: someone stored it.
	t.Run("a pointer to a zero value is not absent", func(t *testing.T) {
		data, err := toJSONB(&testkube.TestWorkflowRerun{})
		require.NoError(t, err)
		assert.JSONEq(t, `{}`, string(data))
	})
}

func TestFromJSONBReadsAnAbsentValueAsNil(t *testing.T) {
	t.Run("an empty column", func(t *testing.T) {
		out, err := fromJSONB[testkube.TestWorkflowRerun](nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	// Rows written before toJSONB stopped emitting it hold this, so reading it
	// as absent is what repairs them.
	t.Run("a literal null", func(t *testing.T) {
		out, err := fromJSONB[testkube.TestWorkflowRerun]([]byte("null"))
		require.NoError(t, err)
		assert.Nil(t, out, "a stored null is an absent value, not an empty one")
	})

	t.Run("a stored value", func(t *testing.T) {
		out, err := fromJSONB[testkube.TestWorkflowRerun]([]byte(`{"executionId":"exec-1","onlyFailed":true}`))
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, "exec-1", out.ExecutionId)
		assert.True(t, out.OnlyFailed)
	})

	t.Run("a stored empty object stays present", func(t *testing.T) {
		out, err := fromJSONB[testkube.TestWorkflowRerun]([]byte(`{}`))
		require.NoError(t, err)
		assert.NotNil(t, out, "an empty object was stored, so it is not absent")
	})
}

// The round trip is what the integration suite asserts: an execution that was
// never a rerun must not come back carrying a rerun policy that names nothing.
func TestJSONBRoundTripKeepsAbsentAbsent(t *testing.T) {
	var absent *testkube.TestWorkflowRerun

	data, err := toJSONB(absent)
	require.NoError(t, err)

	out, err := fromJSONB[testkube.TestWorkflowRerun](data)
	require.NoError(t, err)
	assert.Nil(t, out)
}
