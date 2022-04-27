package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimPatchVersion(t *testing.T) {
	t.Run("trims release version", func(t *testing.T) {
		v := TrimPatchVersion("1.0.3")
		assert.Equal(t, "1.0.0", v)
	})

	t.Run("trims dev version", func(t *testing.T) {
		v := TrimPatchVersion("1.0.3-jd3jd939jd9j")
		assert.Equal(t, "1.0.0-jd3jd939jd9j", v)
	})
}
