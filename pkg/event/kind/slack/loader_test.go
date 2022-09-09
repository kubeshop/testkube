package slack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlackLoader_Load(t *testing.T) {

	t.Run("loads Slack listeners for all event types", func(t *testing.T) {
		// given
		l := NewSlackLoader()

		// when
		listeners, err := l.Load()

		// then
		assert.NoError(t, err)
		assert.Len(t, listeners, 1)
	})

}
