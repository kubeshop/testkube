package api

import "github.com/gofiber/fiber/v2"

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
}

func (s Server) Run() {
	s.Mux.Listen(":3000")
}
