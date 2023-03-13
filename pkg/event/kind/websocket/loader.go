package websocket

import (
	"sync"

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
	mutex    sync.Mutex
	Listener *WebsocketListener
}

func (l *WebsocketLoader) Kind() string {
	return "websocket"
}

func (l *WebsocketLoader) Load() (listeners common.Listeners, err error) {
	return common.Listeners{l.Listener}, nil
}

func (l *WebsocketLoader) Add(conn *websocket.Conn) chan bool {
	var end chan bool
	id := uuid.NewString()

	conn.SetCloseHandler(func(code int, text string) error {
		for i, websocket := range l.Listener.Websockets {
			if websocket.Id == id {
				l.mutex.Lock()
				l.Listener.Websockets = append(l.Listener.Websockets[:i], l.Listener.Websockets[i+1:]...)
				l.mutex.Unlock()
				break
			}
		}

		end <- true
		return nil
	})

	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.Listener.Websockets = append(l.Listener.Websockets, Websocket{Id: id, Conn: conn, Events: testkube.AllEventTypes})

	conn.WriteJSON(map[string]string{"message": "connected to Testkube Events", "id": id})
	return end
}
