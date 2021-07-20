package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var test = SriptsFromCRD{
	name: map[string]string{
		"First":  "firstValue",
		"Second": "secondValue",
		"Third":  "thirdValue",
	},
}

// TestGetCRDEmptiness. Checking if returned Script value is not empty.
func TestGetCRDEmptiness(t *testing.T) {
	ans, _ := test.Get("First")
	assert.NotNil(t, ans)
}

// TestGetCRDCorrectness. Testing if returned value is what is expected.
func TestGetCRDCorrectness(t *testing.T) {
	ans, _ := test.Get("First")
	assert.Equal(t, ans, "firstValue")
}
