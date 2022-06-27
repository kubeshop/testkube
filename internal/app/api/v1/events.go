package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/fluxcd/pkg/runtime/events"
)

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
