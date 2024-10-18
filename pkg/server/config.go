package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Config for HTTP server
type Config struct {
	Port int
	Http fiber.Config
}

// Addr returns port based address
func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
