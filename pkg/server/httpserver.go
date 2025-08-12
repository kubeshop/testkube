package server

import (
	"context"
	"net"
	"net/http"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

// NewServer returns new HTTP server instance, initializes logger and metrics
func NewServer(config Config) HTTPServer {
	config.Http.DisableStartupMessage = true

	s := HTTPServer{
		Mux:    fiber.New(config.Http),
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

	// global log for requests
	s.Mux.Use(func(c *fiber.Ctx) error {
		s.Log.Debugw("request", "method", string(c.Request().Header.Method()), "path", c.Request().URI().String())
		return c.Next()
	})

	s.Mux.Use(pprof.New())

	s.Mux.Use(adaptor.HTTPMiddleware(userAgentMiddleware))

	// server generic endpoints
	s.Mux.Get("/health", s.HealthEndpoint())
	s.Mux.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// v1 API
	v1 := s.Mux.Group("/v1")
	v1.Static("/api-docs", "./api/v1")

	s.Routes = v1

}

// RoutesHandler is a handler to get existing routes
func (s *HTTPServer) RoutesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var routes []fiber.Route

		stack := s.Mux.Stack()
		for _, e := range stack {
			for _, s := range e {
				route := *s
				routes = append(routes, route)
			}
		}

		return c.JSON(routes)
	}
}

// Run starts listening for incoming connetions
func (s *HTTPServer) Run(ctx context.Context) error {
	// Use basic routes
	s.Routes.Get("/routes", s.RoutesHandler())

	// Start server
	l, err := net.Listen("tcp", s.Config.Addr())
	if err != nil {
		return err
	}
	// this function listens for finished context and calls graceful shutdown on the API server
	go func() {
		<-ctx.Done()
		s.Log.Infof("shutting down Testkube API server")
		_ = s.Mux.Shutdown()
	}()
	return s.Mux.Listener(l)
}

// userAgentMiddleware identifies if in request is coming a header "User-Agent" to send event
func userAgentMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		// If userAgent is one of the following values: backstage, cli
		if userAgent == "backstage" || userAgent == "cli" {
			// Send event
			telemetry.SendUserAgentEvent(userAgent)
		}
		next.ServeHTTP(w, r)
	})
}
