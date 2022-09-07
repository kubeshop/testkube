package websocket

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"go.uber.org/zap"
)

var _ common.Listener = &WebsocketListener{}

func NewWebsocketListener(websocket Websocket, selector string, event testkube.EventType) *WebsocketListener {
	return &WebsocketListener{
		Log:       log.DefaultLogger,
		selector:  selector,
		Websocket: websocket,
		event:     event,
	}
}

type WebsocketListener struct {
	Log       *zap.SugaredLogger
	event     testkube.EventType
	Websocket Websocket
	selector  string
}

func (l *WebsocketListener) Name() string {
	return common.ListenerName("websocket." + l.Websocket.Id)
}

func (l *WebsocketListener) Selector() string {
	return l.selector
}

func (l *WebsocketListener) Event() testkube.EventType {
	return l.event
}
func (l *WebsocketListener) Metadata() map[string]string {
	return map[string]string{
		"id": l.Websocket.Conn.Params("id"),
	}
}

func (l *WebsocketListener) Notify(event testkube.Event) (result testkube.EventResult) {
	err := l.Websocket.Conn.WriteJSON(event)
	if err != nil {
		return testkube.NewFailedEventResult(event.Id, err)
	}

	return testkube.NewSuccessEventResult(event.Id, "message-sent to client")
}

func (l *WebsocketListener) Kind() string {
	return "websocket"
}
