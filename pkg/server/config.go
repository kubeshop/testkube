package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Config for HTTP server
type Config struct {
	Port      int
	Fullname  string
	ClusterID string
	Http      fiber.Config
}

// Addr returns port based address
func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
