package v1

import (
	"github.com/gofiber/fiber/v2"
)

// GetDebugListenersHandler returns event logs
func (s TestkubeAPI) GetDebugListenersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(s.Events.Listeners.Log())
	}
}
