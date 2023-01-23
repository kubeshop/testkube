package websocket

import (
	"github.com/gofiber/websocket/v2"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Websocket struct {
	Id       string
	Conn     *websocket.Conn
	Selector string
	Events   []testkube.EventType
}
