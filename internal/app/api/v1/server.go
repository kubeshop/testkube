package v1

import (
	"github.com/kelseyhightower/envconfig"
	scriptscr "github.com/kubeshop/kubetest-operator/client/scripts"
	"github.com/kubeshop/kubetest/internal/pkg/api/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/server"
)

func NewServer(repository result.Repository, scriptsClient scriptscr.ScriptsClient) KubetestAPI {

	var httpConfig server.Config
	envconfig.Process("APISERVER", &httpConfig)

	s := KubetestAPI{
		HTTPServer:    server.NewServer(httpConfig),
		Repository:    repository,
		ScriptsClient: scriptsClient,
		Metrics:       NewMetrics(),
	}

	s.Init()
	return s
}

type KubetestAPI struct {
	server.HTTPServer
	Repository    result.Repository
	ScriptsClient scriptscr.ScriptsClient
	Metrics       Metrics
}

func (s KubetestAPI) Init() {
	s.Routes.Static("/api-docs", "./api/v1")

	scripts := s.Routes.Group("/scripts")
	scripts.Get("/", s.GetAllScripts())
	scripts.Post("/", s.CreateScript())
	scripts.Get("/:id/executions", s.GetScriptExecutions())
	scripts.Post("/:id/executions", s.ExecuteScript())
	scripts.Get("/:id/executions/:executionID", s.GetScriptExecution())
	scripts.Post("/:id/executions/:executionID/abort", s.AbortScriptExecution())
}
