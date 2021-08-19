package server

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/kubtest/pkg/log"
	"github.com/kubeshop/kubtest/pkg/problem"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// NewServer returns new HTTP server instance, initializes logger and metrics
func NewServer(config Config) HTTPServer {
	s := HTTPServer{
		Mux:    fiber.New(),
		Log:    log.DefaultLogger,
		Config: config,
	}

	s.Init()
	return s
}

// HTTPServer represents basic REST HTTP server abstarction
type HTTPServer struct {
	Mux    *fiber.App
	Log    *zap.SugaredLogger
	Routes fiber.Router
	Config Config
}

// Init initializes router and setting up basic routes for health and metrics
func (s *HTTPServer) Init() {
	// server generic endpoints
	s.Mux.Get("/health", s.HealthEndpoint())
	s.Mux.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// v1 API
	v1 := s.Mux.Group("/v1")
	v1.Static("/api-docs", "./api/v1")

	s.Routes = v1
}

// Error writes rfc-7807 json problem to response
func (s *HTTPServer) Error(c *fiber.Ctx, status int, err error) error {
	c.Status(status)
	c.Response().Header.Set("Content-Type", "application/problem+json")
	s.Log.Errorw(err.Error(), "status", status)
	return c.JSON(problem.New(status, err.Error()))
}

// Run starts listening for incoming connetions
func (s HTTPServer) Run() error {
	return s.Mux.Listen(s.Config.Addr())
}
