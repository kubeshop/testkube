package v1

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/kelseyhightower/envconfig"

	executorscr "github.com/kubeshop/testkube-operator/client/executors"
	scriptscr "github.com/kubeshop/testkube-operator/client/scripts"
	testscr "github.com/kubeshop/testkube-operator/client/tests"
	"github.com/kubeshop/testkube/internal/pkg/api"
	"github.com/kubeshop/testkube/internal/pkg/api/datefilter"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func NewServer(
	executionsResults result.Repository,
	testExecutionsResults testresult.Repository,
	scriptsClient *scriptscr.ScriptsClient,
	executorsClient *executorscr.ExecutorsClient,
	testsClient *testscr.TestsClient,
	secretClient *secret.Client,
) TestKubeAPI {

	var httpConfig server.Config
	envconfig.Process("APISERVER", &httpConfig)

	executor, err := client.NewJobExecutor(executionsResults)
	if err != nil {
		panic(err)
	}

	s := TestKubeAPI{
		HTTPServer:           server.NewServer(httpConfig),
		TestExecutionResults: testExecutionsResults,
		ExecutionResults:     executionsResults,
		Executor:             executor,
		ScriptsClient:        scriptsClient,
		ExecutorsClient:      executorsClient,
		SecretClient:         secretClient,
		TestsClient:          testsClient,
		Metrics:              NewMetrics(),
	}

	s.Init()
	return s
}

type TestKubeAPI struct {
	server.HTTPServer
	ExecutionResults     result.Repository
	TestExecutionResults testresult.Repository
	Executor             client.Executor
	TestsClient          *testscr.TestsClient
	ScriptsClient        *scriptscr.ScriptsClient
	ExecutorsClient      *executorscr.ExecutorsClient
	SecretClient         *secret.Client
	Metrics              Metrics
	Storage              storage.Client
	storageParams        storageParams
}

type storageParams struct {
	SSL             bool
	Endpoint        string
	AccessKeyId     string
	SecretAccessKey string
	Location        string
	Token           string
}

func (s TestKubeAPI) Init() {
	envconfig.Process("STORAGE", &s.storageParams)

	s.Storage = minio.NewClient(s.storageParams.Endpoint, s.storageParams.AccessKeyId, s.storageParams.SecretAccessKey, s.storageParams.Location, s.storageParams.Token, s.storageParams.SSL)

	s.Routes.Static("/api-docs", "./api/v1")
	s.Routes.Use(cors.New())

	s.Routes.Get("/info", s.InfoHandler())
	s.Routes.Get("/routes", s.RoutesHandler())

	executors := s.Routes.Group("/executors")

	executors.Post("/", s.CreateExecutorHandler())
	executors.Get("/", s.ListExecutorsHandler())
	executors.Get("/:name", s.GetExecutorHandler())
	executors.Delete("/:name", s.DeleteExecutorHandler())

	executions := s.Routes.Group("/executions")

	executions.Get("/", s.ListExecutionsHandler())
	executions.Get("/:executionID", s.GetExecutionHandler())
	executions.Get("/:executionID/artifacts", s.ListArtifactsHandler())
	executions.Get("/:executionID/logs", s.ExecutionLogsHandler())
	executions.Get("/:executionID/artifacts/:filename", s.GetArtifactHandler())

	scripts := s.Routes.Group("/scripts")

	scripts.Get("/", s.ListScriptsHandler())
	scripts.Post("/", s.CreateScriptHandler())
	scripts.Patch("/:id", s.UpdateScriptHandler())
	scripts.Delete("/", s.DeleteScriptsHandler())

	scripts.Get("/:id", s.GetScriptHandler())
	scripts.Delete("/:id", s.DeleteScriptHandler())

	scripts.Post("/:id/executions", s.ExecuteScriptHandler())

	scripts.Get("/:id/executions", s.ListExecutionsHandler())
	scripts.Get("/:id/executions/:executionID", s.GetExecutionHandler())
	scripts.Delete("/:id/executions/:executionID", s.AbortExecutionHandler())

	tests := s.Routes.Group("/tests")

	tests.Post("/", s.CreateTestHandler())
	tests.Get("/", s.ListTestsHandler())
	tests.Delete("/", s.DeleteTestsHandler())
	tests.Get("/:id", s.GetTestHandler())
	tests.Delete("/:id", s.DeleteTestHandler())

	tests.Post("/:id/executions", s.ExecuteTestHandler())
	tests.Get("/:id/executions", s.ListTestExecutionsHandler())
	tests.Get("/:id/executions/:executionID", s.GetTestExecutionHandler())

	testExecutions := s.Routes.Group("/test-executions")
	testExecutions.Get("/", s.ListTestExecutionsHandler())
	testExecutions.Get("/:executionID", s.GetTestExecutionHandler())

	tags := s.Routes.Group("/tags")
	tags.Get("/", s.ListTagsHandler())

}

func (s TestKubeAPI) InfoHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(testkube.ServerInfo{
			Commit:  api.Commit,
			Version: api.Version,
		})
	}
}

func (s TestKubeAPI) RoutesHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		routes := []fiber.Route{}

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

// TODO should we use single generic filter for all list based resources ?
// currently filters for e.g. scripts are done "by hand"
func getFilterFromRequest(c *fiber.Ctx) result.Filter {

	filter := result.NewExecutionsFilter()

	// id for /scripts/ID/executions
	scriptName := c.Params("id", "")
	if scriptName != "" {
		// query param for /executions?scriptName
		scriptName = c.Query("scriptName", "")
	}
	if scriptName != "" {
		filter = filter.WithScriptName(scriptName)
	}

	textSearch := c.Query("textSearch", "")
	if textSearch != "" {
		filter = filter.WithTextSearch(textSearch)
	}

	page, err := strconv.Atoi(c.Query("page", ""))
	if err == nil {
		filter = filter.WithPage(page)
	}

	pageSize, err := strconv.Atoi(c.Query("pageSize", ""))
	if err == nil && pageSize != 0 {
		filter = filter.WithPageSize(pageSize)
	}

	status := c.Query("status", "")
	if status != "" {
		filter = filter.WithStatus(testkube.ExecutionStatus(status))
	}

	objectType := c.Query("type", "")
	if objectType != "" {
		filter = filter.WithType(objectType)
	}

	dFilter := datefilter.NewDateFilter(c.Query("startDate", ""), c.Query("endDate", ""))
	if dFilter.IsStartValid {
		filter = filter.WithStartDate(dFilter.Start)
	}

	if dFilter.IsEndValid {
		filter = filter.WithEndDate(dFilter.End)
	}

	rawTags := c.Query("tags")
	if rawTags != "" {
		filter = filter.WithTags(strings.Split(rawTags, ","))
	}

	return filter
}
