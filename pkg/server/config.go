package server

import "fmt"

// Config for HTTP server
type Config struct {
	Port      int
	Fullname  string
	ClusterID string
}

// Addr returns port based address
func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
