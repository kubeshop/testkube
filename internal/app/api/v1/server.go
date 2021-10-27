package v1

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/kelseyhightower/envconfig"
	executorscr "github.com/kubeshop/testkube-operator/client/executors"
	scriptscr "github.com/kubeshop/testkube-operator/client/scripts"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/server"
)

func NewServer(repository result.Repository, scriptsClient *scriptscr.ScriptsClient, executorsClient *executorscr.ExecutorsClient) testkubeAPI {

	// TODO consider moving to server pkg as some API_HTTPSERVER_ config prefix
	var httpConfig server.Config
	envconfig.Process("APISERVER", &httpConfig)

	// TODO remove it when executor CRD will be fully implemented
	var executorClientConfig client.RestExecutorConfig
	envconfig.Process("POSTMANEXECUTOR", &executorClientConfig)

	s := testkubeAPI{
		HTTPServer:      server.NewServer(httpConfig),
		Repository:      repository,
		ScriptsClient:   scriptsClient,
		ExecutorsClient: executorsClient,
		Metrics:         NewMetrics(),
		Executors:       client.NewExecutors(executorsClient),
	}

	s.Init()
	return s
}

type testkubeAPI struct {
	server.HTTPServer
	Repository      result.Repository
	Executors       client.Executors
	ScriptsClient   *scriptscr.ScriptsClient
	ExecutorsClient *executorscr.ExecutorsClient
	Metrics         Metrics
}

func (s testkubeAPI) Init() {
	s.Routes.Static("/api-docs", "./api/v1")

	s.Routes.Get("/info", s.Info())

	executors := s.Routes.Group("/executors")

	executors.Post("/", s.CreateExecutor())
	executors.Get("/", s.ListExecutors())
	executors.Get("/:name", s.GetExecutor())
	executors.Delete("/:name", s.DeleteExecutor())

	executions := s.Routes.Group("/executions")
	executions.Use(cors.New())

	executions.Get("/", s.ListExecutions())
	executions.Get("/:executionID", s.GetExecution())

	scripts := s.Routes.Group("/scripts")

	scripts.Get("/", s.ListScripts())
	scripts.Get("/:id", s.GetScript())
	scripts.Post("/", s.CreateScript())

	scripts.Post("/:id/executions", s.ExecuteScript())
	scripts.Post("/:id/executions/:executionID/abort", s.AbortExecution())

	scripts.Get("/:id/executions", s.ListExecutions())
	scripts.Get("/:id/executions/:executionID", s.GetExecution())
}
