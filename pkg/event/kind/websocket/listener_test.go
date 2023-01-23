package websocket

import (
	"testing"

	"github.com/gofiber/websocket/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestWebsocketListener(t *testing.T) {
	t.Skip("not implemented")

	// given
	l := NewWebsocketListener()
	l.Websockets = []Websocket{{
		Id:   "1",
		Conn: &websocket.Conn{},
	}}

	// when
	result := l.Notify(testkube.NewEventStartTest(testkube.NewQueuedExecution()))

	// then
	assert.Equal(t, "", result.Error_)
}
