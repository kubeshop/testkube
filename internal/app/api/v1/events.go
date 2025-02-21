package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	events "github.com/fluxcd/pkg/apis/event/v1beta1"
)

func (s *TestkubeAPI) EventsStreamHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		s.Log.Debugw("handling websocket connection", "id", c.Params("id"), "remoteAddr", c.RemoteAddr(), "localAddr", c.LocalAddr())

		// wait for disconnect
		// WebsocketLoader will add WebsocketListener which will send data to `c`
		<-s.WebsocketLoader.Add(c)

		s.Log.Debugw("websocket closed", "id", c.Params("id"))
	})
}

// GetTestHandler is method for getting an existing test
func (s *TestkubeAPI) FluxEventHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := c.Body()

		var event events.Event
		err := json.Unmarshal(body, &event)
		if err != nil {
			return s.Error(c, http.StatusBadRequest, err)
		}

		// TODO handle Flux logic on deployment change
		// check event.InvolvedObject?
		switch event.Reason {
		default:
			s.Log.Debugw("got FluxCD event", "event", event)
		}

		return c.JSON(event)
	}
}
