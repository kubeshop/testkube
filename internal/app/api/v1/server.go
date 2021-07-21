package v1

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/kubetest/pkg/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func NewServer() Server {
	s := Server{
		Mux: fiber.New(),
		Log: log.DefaultLogger,
	}

	s.Init()
	return s
}

type Server struct {
	Mux *fiber.App
	Log *zap.SugaredLogger
}

func (s Server) Init() {
	// server generic endpoints
	s.Mux.Get("/health", s.HealthEndpoint())
	s.Mux.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// v1 API
	v1 := s.Mux.Group("/v1")
	v1.Static("/api-docs", "./api/v1")

	scripts := v1.Group("/scripts")
	scripts.Get("/", s.GetAllScripts())
	scripts.Get("/:id/executions", s.GetAllScriptExecutions())
	scripts.Post("/:id/executions", s.ExecuteScript())
	scripts.Get("/:id/executions/:executionID", s.GetScriptExecution())
	scripts.Post("/:id/executions/:executionID/abort", s.AbortScriptExecution())
}

func (s Server) Run() {
	s.Mux.Listen(":8080")
}
