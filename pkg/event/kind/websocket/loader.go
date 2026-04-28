package websocket

import (
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

func NewWebsocketLoader() *WebsocketLoader {
	return &WebsocketLoader{
		Listener: NewWebsocketListener(),
	}
}

type WebsocketLoader struct {
	Listener *WebsocketListener
}

func (l *WebsocketLoader) Kind() string {
	return "websocket"
}

func (l *WebsocketLoader) Load() (listeners common.Listeners, err error) {
	return common.Listeners{l.Listener}, nil
}

func (l *WebsocketLoader) Add(conn *websocket.Conn) chan bool {
	end := make(chan bool, 1)
	id := uuid.NewString()

	conn.SetCloseHandler(func(code int, text string) error {
		l.Listener.RemoveWebsocket(id)

		end <- true
		return nil
	})

	ws := &Websocket{Id: id, Conn: conn, Events: testkube.AllEventTypes}
	l.Listener.AddWebsocket(ws)

	_ = ws.SendJSON(map[string]string{"message": "connected to Testkube Events", "id": id})
	return end
}
