package slack

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestSlackLoader_Load(t *testing.T) {
	t.Parallel()
	t.Run("loads Slack listeners for all event types", func(t *testing.T) {
		t.Parallel()
		// given
		// default slack notifier is not ready by default
		l := NewSlackLoader("", "", "", "", testkube.AllEventTypes, nil)

		// when
		listeners, err := l.Load()

		// then
		assert.NoError(t, err)
		assert.Len(t, listeners, 0)
	})

}
