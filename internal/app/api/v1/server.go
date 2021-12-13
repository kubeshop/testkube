package v1

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/kelseyhightower/envconfig"
	executorscr "github.com/kubeshop/testkube-operator/client/executors"
	scriptscr "github.com/kubeshop/testkube-operator/client/scripts"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func NewServer(repository result.Repository, scriptsClient *scriptscr.ScriptsClient, executorsClient *executorscr.ExecutorsClient) testkubeAPI {

	// TODO consider moving to server pkg as some API_HTTPSERVER_ config prefix
	var httpConfig server.Config
	envconfig.Process("APISERVER", &httpConfig)

	executor, err := client.NewJobExecutor(repository)
	if err != nil {
		panic(err)
	}

	s := testkubeAPI{
		HTTPServer:      server.NewServer(httpConfig),
		Repository:      repository,
		Executor:        executor,
		ScriptsClient:   scriptsClient,
		ExecutorsClient: executorsClient,
		Metrics:         NewMetrics(),
	}

	s.Init()
	return s
}

type testkubeAPI struct {
	server.HTTPServer
	Repository      result.Repository
	Executor        client.Executor
	TestsClient     *scriptscr.ScriptsClient
	ScriptsClient   *scriptscr.ScriptsClient
	ExecutorsClient *executorscr.ExecutorsClient
	Metrics         Metrics
	Storage         storage.Client
	storageParams   storageParams
}

type storageParams struct {
	SSL             bool
	Endpoint        string
	AccessKeyId     string
	SecretAccessKey string
	Location        string
	Token           string
}

func (s testkubeAPI) Init() {
	envconfig.Process("STORAGE", &s.storageParams)

	s.Storage = minio.NewClient(s.storageParams.Endpoint, s.storageParams.AccessKeyId, s.storageParams.SecretAccessKey, s.storageParams.Location, s.storageParams.Token, s.storageParams.SSL)

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
	executions.Get("/:executionID/artifacts", s.ListArtifacts())
	executions.Get("/:executionID/logs", s.ExecutionLogs())
	executions.Get("/:executionID/artifacts/:filename", s.GetArtifact())

	scripts := s.Routes.Group("/scripts")

	scripts.Get("/", s.ListScripts())
	scripts.Post("/", s.CreateScript())
	scripts.Patch("/:id", s.UpdateScript())
	scripts.Delete("/", s.DeleteScripts())

	scripts.Get("/:id", s.GetScript())
	scripts.Delete("/:id", s.DeleteScript())

	scripts.Post("/:id/executions", s.ExecuteScript())

	scripts.Get("/:id/executions", s.ListExecutions())
	scripts.Get("/:id/executions/:executionID", s.GetExecution())
	scripts.Delete("/:id/executions/:executionID", s.AbortExecution())

	tests := s.Routes.Group("/scripts")

	tests.Get("/", s.ListTests())
	tests.Get("/:id", s.GetTest())

}
