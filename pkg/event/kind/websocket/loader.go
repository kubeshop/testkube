package websocket

import (
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

func NewWebsocketLoader() *WebsocketLoader {
	return &WebsocketLoader{}
}

type WebsocketLoader struct {
	mutex      sync.Mutex
	Websockets []Websocket
}

func (l *WebsocketLoader) Kind() string {
	return "websocket"
}

func (l *WebsocketLoader) Load() (listeners common.Listeners, err error) {
	return common.Listeners{NewWebsocketListener(l.Websockets)}, nil
}

func (l *WebsocketLoader) Add(conn *websocket.Conn) chan bool {
	var end chan bool
	id := uuid.NewString()

	conn.SetCloseHandler(func(code int, text string) error {
		for i, websocket := range l.Websockets {
			if websocket.Id == id {
				l.mutex.Lock()
				l.Websockets = append(l.Websockets[:i], l.Websockets[i+1:]...)
				l.mutex.Unlock()
				break
			}
		}

		end <- true
		return nil
	})

	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.Websockets = append(l.Websockets, Websocket{Id: id, Conn: conn, Events: testkube.AllEventTypes})
	return end
}
