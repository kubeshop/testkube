package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGA4EventName(t *testing.T) {

	t.Run("filter invalid name", func(t *testing.T) {
		in := "/v1/api/some/api-test"
		out := GAEventName(in)

		assert.Equal(t, "v1_api_some_api_test", out)
	})

	t.Run("filter invalid name above 40 chars", func(t *testing.T) {
		in := "/v1/api/some/api-test-above-40-characters/above-40-chars"
		out := GAEventName(in)

		assert.Equal(t, 40, len(out))
		assert.Equal(t, "v1_api_some_api_test_above_40_characters", out)
	})
}
