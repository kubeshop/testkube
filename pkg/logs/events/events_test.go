package events

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLogFromBytes(t *testing.T) {
	assert := require.New(t)

	t.Run("log line with timestamp passed from kube api", func(t *testing.T) {
		b := []byte("2024-03-11T10:47:41.070097107Z Line")

		l := NewLogFromBytes(b)

		assert.Equal("2024-03-11 10:47:41.070097107 +0000 UTC", l.Time.String())
		assert.Equal("Line", l.Content)
	})

	t.Run("log line without timestamp", func(t *testing.T) {
		b := []byte("Line")

		l := NewLogFromBytes(b)

		assert.Equal("Line", l.Content)
	})

	t.Run("old log line without timestamp", func(t *testing.T) {
		b := []byte(`{"content":"Line"}`)

		l := NewLogFromBytes(b)

		assert.Equal("Line", l.Content)
	})

}
