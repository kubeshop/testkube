package websocket

import (
	"testing"

	"github.com/gofiber/websocket/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestWebsocketListener(t *testing.T) {
	t.Skip("not implemented")

	// given
	ws := Websocket{
		Id:   "1",
		Conn: &websocket.Conn{},
	}

	l := NewWebsocketListener(ws, "", []testkube.TestkubeEventType{})

	// when
	result := l.Notify(testkube.NewTestkubeEventStartTest(testkube.NewQueuedExecution()))

	// then
	assert.Equal(t, "", result.Error_)
}
