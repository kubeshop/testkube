package v1

import (
	"github.com/kelseyhightower/envconfig"
	executorscr "github.com/kubeshop/kubtest-operator/client/executors"
	scriptscr "github.com/kubeshop/kubtest-operator/client/scripts"
	"github.com/kubeshop/kubtest/internal/pkg/api/repository/result"
	"github.com/kubeshop/kubtest/pkg/executor/client"
	"github.com/kubeshop/kubtest/pkg/server"
)

func NewServer(repository result.Repository, scriptsClient *scriptscr.ScriptsClient, executorsClient *executorscr.ExecutorsClient) kubtestAPI {

	// TODO consider moving to server pkg as some API_HTTPSERVER_ config prefix
	var httpConfig server.Config
	envconfig.Process("APISERVER", &httpConfig)

	// TODO remove it when executor CRD will be fully implemented
	var executorClientConfig client.RestExecutorConfig
	envconfig.Process("POSTMANEXECUTOR", &executorClientConfig)

	s := kubtestAPI{
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

type kubtestAPI struct {
	server.HTTPServer
	Repository      result.Repository
	Executors       client.Executors
	ScriptsClient   *scriptscr.ScriptsClient
	ExecutorsClient *executorscr.ExecutorsClient
	Metrics         Metrics
}

func (s kubtestAPI) Init() {
	s.Routes.Static("/api-docs", "./api/v1")

	executions := s.Routes.Group("/executions")

	executions.Get("/", s.ListExecutions())
	executions.Get("/:executionID", s.GetScriptExecution())

	scripts := s.Routes.Group("/scripts")

	scripts.Get("/", s.ListScripts())
	scripts.Get("/:id", s.GetScript())
	scripts.Post("/", s.CreateScript())

	scripts.Post("/:id/executions", s.ExecuteScript())
	scripts.Post("/:id/executions/:executionID/abort", s.AbortExecution())

	scripts.Get("/:id/executions", s.ListExecutions())
	scripts.Get("/:id/executions/:executionID", s.GetScriptExecution())
}
