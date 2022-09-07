package websocket

import (
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

func NewWebsocketLoader() *WebsocketLoader {
	return &WebsocketLoader{}
}

type WebsocketLoader struct {
	mutex      sync.Mutex
	Websockets []Websocket
}

func (r *WebsocketLoader) Kind() string {
	return "websocket"
}

func (r *WebsocketLoader) Load() (listeners common.Listeners, err error) {
	for _, ws := range r.Websockets {
		for _, t := range ws.Events {
			wh := NewWebsocketListener(ws, ws.Selector, t)
			listeners = append(listeners, wh)
		}
	}

	return listeners, nil
}

func (r *WebsocketLoader) Add(conn *websocket.Conn) chan bool {
	var end chan bool
	id := uuid.NewString()

	conn.SetCloseHandler(func(code int, text string) error {
		for i, websocket := range r.Websockets {
			if websocket.Id == id {
				r.mutex.Lock()
				r.Websockets = append(r.Websockets[:i], r.Websockets[i+1:]...)
				r.mutex.Unlock()
				break
			}
		}

		end <- true
		return nil
	})

	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.Websockets = append(r.Websockets, Websocket{Id: id, Conn: conn})
	return end
}
