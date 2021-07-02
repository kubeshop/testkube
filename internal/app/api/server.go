package api

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewServer() Server {
	s := Server{
		Mux: fiber.New(),
	}

	s.Init()
	return s
}

type Server struct {
	Mux *fiber.App
}

func (s Server) Init() {
	s.Mux.Get("/health", s.HealthEndpoint())
	s.Mux.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}

func (s Server) Run() {
	s.Mux.Listen(":3000")
}
