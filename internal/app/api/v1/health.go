package v1

import "github.com/gofiber/fiber/v2"

func (s Server) HealthEndpoint() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("OK 👋!")
	}
}
