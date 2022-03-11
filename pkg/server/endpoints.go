package server

import "github.com/gofiber/fiber/v2"

// HealthEndpoint for health checks
func (s HTTPServer) HealthEndpoint() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("OK 👋!")
	}
}
