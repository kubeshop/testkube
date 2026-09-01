package executiondata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithheldMarker(t *testing.T) {
	t.Run("names the output and the workflow that produced it", func(t *testing.T) {
		marker := WithheldMarker("producer", "token")
		assert.Equal(t, "<testkube:withheld output token of workflow producer>", marker)
		assert.Equal(t, []string{marker}, WithheldMarkersIn(marker))
	})

	t.Run("names just the output when the workflow is unknown", func(t *testing.T) {
		assert.Equal(t, "<testkube:withheld output token>", WithheldMarker("", "token"))
	})

	t.Run("an ordinary value is not a marker", func(t *testing.T) {
		assert.Empty(t, WithheldMarkersIn("abc123"))
		assert.Empty(t, WithheldMarkersIn("<testkube:something-else>"))
	})
}

func TestWithheldMarkersIn(t *testing.T) {
	marker := WithheldMarker("producer", "token")

	t.Run("finds a marker nested anywhere in a value", func(t *testing.T) {
		value := map[string]interface{}{
			"config": map[string]interface{}{"token": "Bearer " + marker},
			"args":   []interface{}{"--flag", 7, nil},
		}
		assert.Equal(t, []string{marker}, WithheldMarkersIn(value))
	})

	t.Run("reports each marker once", func(t *testing.T) {
		other := WithheldMarker("producer", "password")
		value := []interface{}{marker, other, marker}
		assert.Equal(t, []string{marker, other}, WithheldMarkersIn(value))
	})

	t.Run("reports nothing for a value carrying no marker", func(t *testing.T) {
		assert.Empty(t, WithheldMarkersIn(map[string]interface{}{"token": "abc123"}))
		assert.Empty(t, WithheldMarkersIn(nil))
		assert.Empty(t, WithheldMarkersIn("plain"))
	})

	t.Run("finds the marker a struct carries", func(t *testing.T) {
		// The specification handed to another workflow is a struct, and json escapes the
		// angle brackets of the marker - so the walk has to look at the decoded strings,
		// not at the encoded document.
		value := struct {
			Name   string            `json:"name"`
			Config map[string]string `json:"config"`
		}{Name: "consumer", Config: map[string]string{"token": marker}}
		assert.Equal(t, []string{marker}, WithheldMarkersIn(value))
	})

	t.Run("finds the marker an execution publishes as an output", func(t *testing.T) {
		execution := Execution{Id: "exec-1", Workflow: "producer", Outputs: map[string]string{"token": marker}}
		assert.Equal(t, []string{marker}, WithheldMarkersIn(execution.AsMap()))
	})
}

func TestWithheldError(t *testing.T) {
	err := WithheldError("this execution", []string{WithheldMarker("producer", "token")})
	assert.ErrorContains(t, err, "this execution reads an output that was not published")
	assert.ErrorContains(t, err, "<testkube:withheld output token of workflow producer>")
	assert.ErrorContains(t, err, "read_artifact()")
}
