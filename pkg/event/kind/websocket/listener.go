package websocket

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
)

var _ common.Listener = &WebsocketListener{}

func NewWebsocketListener(websocket Websocket, selector string, events []testkube.TestkubeEventType) *WebsocketListener {
	return &WebsocketListener{
		Log:       log.DefaultLogger,
		selector:  selector,
		Websocket: websocket,
		events:    events,
	}
}

type WebsocketListener struct {
	Log       *zap.SugaredLogger
	events    []testkube.TestkubeEventType
	Websocket Websocket
	selector  string
}

func (l *WebsocketListener) Selector() string {
	return l.selector
}

func (l *WebsocketListener) Events() []testkube.TestkubeEventType {
	return l.events
}
func (l *WebsocketListener) Metadata() map[string]string {
	return map[string]string{
		"id": l.Websocket.Conn.Params("id"),
	}
}

func (l *WebsocketListener) Notify(event testkube.TestkubeEvent) (result testkube.TestkubeEventResult) {
	err := l.Websocket.Conn.WriteJSON(event)
	if err != nil {
		return testkube.NewFailedTestkubeEventResult(event.Id, err)
	}

	return testkube.NewSuccessTestkubeEventResult(event.Id, "message-sent to client")
}

func (l *WebsocketListener) Kind() string {
	return "websocket"
}
