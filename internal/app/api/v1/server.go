package v1

import (
	"github.com/kelseyhightower/envconfig"
	scriptscr "github.com/kubeshop/kubetest-operator/client/scripts"
	"github.com/kubeshop/kubetest/internal/pkg/api/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/server"
	"github.com/kubeshop/kubetest/pkg/executor/client"
)

func NewServer(repository result.Repository, scriptsClient scriptscr.ScriptsClient) KubetestAPI {

	// TODO consider moving to server pkg as some API_HTTPSERVER_ config prefix
	var httpConfig server.Config
	envconfig.Process("APISERVER", &httpConfig)

	// TODO remove it when executor CRD will be fully implemented
	var executorClientConfig client.Config
	envconfig.Process("POSTMANEXECUTOR", &executorClientConfig)

	s := KubetestAPI{
		HTTPServer:     server.NewServer(httpConfig),
		Repository:     repository,
		ScriptsClient:  scriptsClient,
		Metrics:        NewMetrics(),
		ExecutorClient: client.NewHTTPExecutorClient(executorClientConfig),
	}

	s.Init()
	return s
}

type KubetestAPI struct {
	server.HTTPServer
	Repository     result.Repository
	ScriptsClient  scriptscr.ScriptsClient
	Metrics        Metrics
	ExecutorClient client.HTTPExecutorClient
}

func (s KubetestAPI) Init() {
	s.Routes.Static("/api-docs", "./api/v1")

	scripts := s.Routes.Group("/scripts")

	scripts.Get("/", s.ListScripts())
	scripts.Get("/:id", s.GetScript())
	scripts.Post("/", s.CreateScript())

	scripts.Post("/:id/executions", s.ExecuteScript())
	scripts.Post("/:id/executions/:executionID/abort", s.AbortScriptExecution())

	scripts.Get("/:id/executions", s.ListExecutions())
	scripts.Get("/:id/executions/:executionID", s.GetScriptExecution())
}
