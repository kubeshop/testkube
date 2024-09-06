package testworkflows

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetetTimestampLength(t *testing.T) {
	t.Run("returns length of nano for valid timestamps", func(t *testing.T) {
		l := getTimestampLength("2006-01-02T15:04:05.999999999Z07:00")
		assert.Equal(t, len(time.RFC3339Nano), l)

		l = getTimestampLength("2006-01-02T15:04:05.999999999+07:00")
		assert.Equal(t, len(time.RFC3339Nano), l)
	})

	t.Run("returns 0 for invalid timestamps", func(t *testing.T) {
		l := getTimestampLength("2006-01-02T15:04:05.99")
		assert.Equal(t, 0, l)

		l = getTimestampLength("2006-01-02T15:04:05.99")
		assert.Equal(t, 0, l)
	})
}
