package websocket

import (
	"sync"

	"github.com/gofiber/websocket/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Websocket struct {
	Id       string
	Conn     *websocket.Conn
	Selector string
	Events   []testkube.EventType
	mu       sync.Mutex
}

func (w *Websocket) SendJSON(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Conn.WriteJSON(v)
}
