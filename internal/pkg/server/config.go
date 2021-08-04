package server

import "fmt"

type Config struct {
	Port int
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
