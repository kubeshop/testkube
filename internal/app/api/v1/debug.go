package v1

import (
	"github.com/gofiber/fiber/v2"
)

// GetConfigsHandler returns configuration
func (s TestkubeAPI) GetDebugListenersHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(s.Events.Listeners.Log())
	}
}
