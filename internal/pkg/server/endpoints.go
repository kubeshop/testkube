package server

import "github.com/gofiber/fiber/v2"

func (s HTTPServer) HealthEndpoint() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("OK 👋!")
	}
}
