package triggers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLabelsToSelector(t *testing.T) {
	t.Parallel()

	singleLabel := map[string]string{"app": "test"}
	expected := "app=test"
	result := labelsToSelector(singleLabel)
	assert.Equal(t, expected, result)

	multiLabel := map[string]string{"app": "test", "tier": "backend"}
	expected = "app=test,tier=backend"
	result = labelsToSelector(multiLabel)
	assert.Equal(t, expected, result)
}
