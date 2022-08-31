package v1

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/fluxcd/pkg/runtime/events"
)

func (s TestkubeAPI) EventsTestHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		e := testkube.NewQueuedExecution()
		e.Id = "test-execution-id"
		s.Events.Notify(testkube.NewTestkubeEventStartTest(e))
		return nil
	}
}

func (s TestkubeAPI) EventsStreamHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// c.Locals is added to the *websocket.Conn
		log.Println(c.Locals("allowed"))  // true
		log.Println(c.Params("id"))       // 123
		log.Println(c.Query("v"))         // 1.0
		log.Println(c.Cookies("session")) // ""

		<-s.WebsocketLoader.Add(c)
		s.Log.Debugw("websocket closed", "id", c.Params("id"))
	})
}

// GetTestHandler is method for getting an existing test
func (s TestkubeAPI) FluxEventHandler() fiber.Handler {
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
