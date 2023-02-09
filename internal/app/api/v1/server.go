package v1

import (
	"context"
	"encoding/base64"
	"os"
	"strconv"
	"time"

	"github.com/kubeshop/testkube/pkg/version"

	"github.com/kubeshop/testkube/pkg/datefilter"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"

	"k8s.io/client-go/kubernetes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/kelseyhightower/envconfig"

	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsourcesclientv1 "github.com/kubeshop/testkube-operator/client/testsources/v1"
	testsuitesclientv2 "github.com/kubeshop/testkube-operator/client/testsuites/v2"
	testkubeclientset "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/config"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/kind/slack"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	kubeexecutor "github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/oauth"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

const HeartbeatInterval = time.Hour

func NewTestkubeAPI(
	namespace string,
	testExecutionResults result.Repository,
	testsuiteExecutionsResults testresult.Repository,
	testsClient *testsclientv3.TestsClient,
	executorsClient *executorsclientv1.ExecutorsClient,
	testsuitesClient *testsuitesclientv2.TestSuitesClient,
	secretClient *secret.Client,
	webhookClient *executorsclientv1.WebhooksClient,
	clientset kubernetes.Interface,
	testkubeClientset testkubeclientset.Interface,
	testsourcesClient *testsourcesclientv1.TestSourcesClient,
	configMap config.Repository,
	clusterId string,
	eventsEmitter *event.Emitter,
	executor client.Executor,
	containerExecutor client.Executor,
	metrics metrics.Metrics,
	templates kubeexecutor.Templates,
	scheduler *scheduler.Scheduler,
) TestkubeAPI {

	var httpConfig server.Config
	err := envconfig.Process("APISERVER", &httpConfig)
	// Do we want to panic here or just ignore the error
	if err != nil {
		panic(err)
	}

	httpConfig.ClusterID = clusterId

	s := TestkubeAPI{
		HTTPServer:           server.NewServer(httpConfig),
		TestExecutionResults: testsuiteExecutionsResults,
		ExecutionResults:     testExecutionResults,
		TestsClient:          testsClient,
		ExecutorsClient:      executorsClient,
		SecretClient:         secretClient,
		Clientset:            clientset,
		TestsSuitesClient:    testsuitesClient,
		TestKubeClientset:    testkubeClientset,
		Metrics:              metrics,
		Events:               eventsEmitter,
		WebhooksClient:       webhookClient,
		TestSourcesClient:    testsourcesClient,
		Namespace:            namespace,
		ConfigMap:            configMap,
		Executor:             executor,
		ContainerExecutor:    containerExecutor,
		templates:            templates,
		scheduler:            scheduler,
	}

	// will be reused in websockets handler
	s.WebsocketLoader = ws.NewWebsocketLoader()

	s.Events.Loader.Register(webhook.NewWebhookLoader(webhookClient))
	s.Events.Loader.Register(s.WebsocketLoader)
	s.Events.Loader.Register(s.getSlackLoader())

	s.InitEnvs()
	s.InitStorage()
	s.InitRoutes()

	return s
}

type TestkubeAPI struct {
	server.HTTPServer
	ExecutionResults     result.Repository
	TestExecutionResults testresult.Repository
	Executor             client.Executor
	ContainerExecutor    client.Executor
	TestsSuitesClient    *testsuitesclientv2.TestSuitesClient
	TestsClient          *testsclientv3.TestsClient
	ExecutorsClient      *executorsclientv1.ExecutorsClient
	SecretClient         *secret.Client
	WebhooksClient       *executorsclientv1.WebhooksClient
	TestKubeClientset    testkubeclientset.Interface
	TestSourcesClient    *testsourcesclientv1.TestSourcesClient
	Metrics              metrics.Metrics
	Storage              storage.Client
	storageParams        storageParams
	Namespace            string
	oauthParams          oauthParams
	WebsocketLoader      *ws.WebsocketLoader
	Events               *event.Emitter
	ConfigMap            config.Repository
	templates            kubeexecutor.Templates
	scheduler            *scheduler.Scheduler
	Clientset            kubernetes.Interface
}

type storageParams struct {
	SSL             bool
	Endpoint        string
	AccessKeyId     string
	SecretAccessKey string
	Location        string
	Token           string
	Bucket          string
}

type oauthParams struct {
	ClientID     string
	ClientSecret string
	Provider     oauth.ProviderType
	Scopes       string
}

// SendTelemetryStartEvent sends anonymous start event to telemetry trackers
func (s TestkubeAPI) SendTelemetryStartEvent(ctx context.Context) {
	telemetryEnabled, err := s.ConfigMap.GetTelemetryEnabled(ctx)
	if err != nil {
		s.Log.Errorw("error getting config map", "error", err)
	}

	if !telemetryEnabled {
		return
	}

	out, err := telemetry.SendServerStartEvent(s.Config.ClusterID, version.Version)
	if err != nil {
		s.Log.Debug("telemetry send error", "error", err.Error())
	} else {
		s.Log.Debugw("sending telemetry server start event", "output", out)
	}
}

// Init initializes api server settings
func (s *TestkubeAPI) InitEnvs() {
	if err := envconfig.Process("STORAGE", &s.storageParams); err != nil {
		s.Log.Infow("Processing STORAGE environment config", err)
	}

	if err := envconfig.Process("TESTKUBE_OAUTH", &s.oauthParams); err != nil {
		s.Log.Infow("Processing TESTKUBE_OAUTH environment config", err)
	}
}

func (s *TestkubeAPI) InitStorage() {
	s.Storage = minio.NewClient(s.storageParams.Endpoint,
		s.storageParams.AccessKeyId,
		s.storageParams.SecretAccessKey,
		s.storageParams.Location,
		s.storageParams.Token,
		s.storageParams.Bucket,
		s.storageParams.SSL)
}

func (s *TestkubeAPI) InitRoutes() {
	s.Routes.Static("/api-docs", "./api/v1")
	s.Routes.Use(cors.New())
	s.Routes.Use(s.AuthHandler())

	s.Routes.Get("/info", s.InfoHandler())
	s.Routes.Get("/routes", s.RoutesHandler())
	s.Routes.Get("/debug", s.DebugHandler())

	executors := s.Routes.Group("/executors")

	executors.Post("/", s.CreateExecutorHandler())
	executors.Get("/", s.ListExecutorsHandler())
	executors.Get("/:name", s.GetExecutorHandler())
	executors.Patch("/:name", s.UpdateExecutorHandler())
	executors.Delete("/:name", s.DeleteExecutorHandler())
	executors.Delete("/", s.DeleteExecutorsHandler())

	webhooks := s.Routes.Group("/webhooks")

	webhooks.Post("/", s.CreateWebhookHandler())
	webhooks.Get("/", s.ListWebhooksHandler())
	webhooks.Get("/:name", s.GetWebhookHandler())
	webhooks.Delete("/:name", s.DeleteWebhookHandler())
	webhooks.Delete("/", s.DeleteWebhooksHandler())

	executions := s.Routes.Group("/executions")

	executions.Get("/", s.ListExecutionsHandler())
	executions.Post("/", s.ExecuteTestsHandler())
	executions.Get("/:executionID", s.GetExecutionHandler())
	executions.Get("/:executionID/artifacts", s.ListArtifactsHandler())
	executions.Get("/:executionID/logs", s.ExecutionLogsHandler())
	executions.Get("/:executionID/logs/stream", s.ExecutionLogsStreamHandler())
	executions.Get("/:executionID/artifacts/:filename", s.GetArtifactHandler())

	tests := s.Routes.Group("/tests")

	tests.Get("/", s.ListTestsHandler())
	tests.Post("/", s.CreateTestHandler())
	tests.Patch("/:id", s.UpdateTestHandler())
	tests.Delete("/", s.DeleteTestsHandler())

	tests.Get("/:id", s.GetTestHandler())
	tests.Delete("/:id", s.DeleteTestHandler())

	tests.Get("/:id/metrics", s.TestMetricsHandler())

	tests.Post("/:id/executions", s.ExecuteTestsHandler())

	tests.Get("/:id/executions", s.ListExecutionsHandler())
	tests.Get("/:id/executions/:executionID", s.GetExecutionHandler())
	tests.Patch("/:id/executions/:executionID", s.AbortExecutionHandler())

	testWithExecutions := s.Routes.Group("/test-with-executions")
	testWithExecutions.Get("/", s.ListTestWithExecutionsHandler())
	testWithExecutions.Get("/:id", s.GetTestWithExecutionHandler())

	testsuites := s.Routes.Group("/test-suites")

	testsuites.Post("/", s.CreateTestSuiteHandler())
	testsuites.Patch("/:id", s.UpdateTestSuiteHandler())
	testsuites.Get("/", s.ListTestSuitesHandler())
	testsuites.Delete("/", s.DeleteTestSuitesHandler())
	testsuites.Get("/:id", s.GetTestSuiteHandler())
	testsuites.Delete("/:id", s.DeleteTestSuiteHandler())

	testsuites.Post("/:id/executions", s.ExecuteTestSuitesHandler())
	testsuites.Get("/:id/executions", s.ListTestSuiteExecutionsHandler())
	testsuites.Get("/:id/executions/:executionID", s.GetTestSuiteExecutionHandler())
	testsuites.Get("/:id/executions/:executionID/artifacts", s.ListTestSuiteArtifactsHandler())
	testsuites.Patch("/:id/executions/:executionID", s.AbortTestSuiteExecutionHandler())

	testsuites.Get("/:id/tests", s.ListTestSuiteTestsHandler())

	testsuites.Get("/:id/metrics", s.TestSuiteMetricsHandler())

	testSuiteExecutions := s.Routes.Group("/test-suite-executions")
	testSuiteExecutions.Get("/", s.ListTestSuiteExecutionsHandler())
	testSuiteExecutions.Post("/", s.ExecuteTestSuitesHandler())
	testSuiteExecutions.Get("/:executionID", s.GetTestSuiteExecutionHandler())
	testSuiteExecutions.Get("/:executionID/artifacts", s.ListTestSuiteArtifactsHandler())
	testSuiteExecutions.Patch("/:executionID", s.AbortTestSuiteExecutionHandler())

	testSuiteWithExecutions := s.Routes.Group("/test-suite-with-executions")
	testSuiteWithExecutions.Get("/", s.ListTestSuiteWithExecutionsHandler())
	testSuiteWithExecutions.Get("/:id", s.GetTestSuiteWithExecutionHandler())

	testTriggers := s.Routes.Group("/triggers")
	testTriggers.Get("/", s.ListTestTriggersHandler())
	testTriggers.Post("/", s.CreateTestTriggerHandler())
	testTriggers.Patch("/", s.BulkUpdateTestTriggersHandler())
	testTriggers.Delete("/", s.DeleteTestTriggersHandler())
	testTriggers.Get("/:id", s.GetTestTriggerHandler())
	testTriggers.Patch("/:id", s.UpdateTestTriggerHandler())
	testTriggers.Delete("/:id", s.DeleteTestTriggerHandler())

	keymap := s.Routes.Group("/keymap")
	keymap.Get("/triggers", s.GetTestTriggerKeyMapHandler())

	testsources := s.Routes.Group("/test-sources")
	testsources.Post("/", s.CreateTestSourceHandler())
	testsources.Get("/", s.ListTestSourcesHandler())
	testsources.Patch("/", s.ProcessTestSourceBatchHandler())
	testsources.Get("/:name", s.GetTestSourceHandler())
	testsources.Patch("/:name", s.UpdateTestSourceHandler())
	testsources.Delete("/:name", s.DeleteTestSourceHandler())
	testsources.Delete("/", s.DeleteTestSourcesHandler())

	labels := s.Routes.Group("/labels")
	labels.Get("/", s.ListLabelsHandler())

	slack := s.Routes.Group("/slack")
	slack.Get("/", s.OauthHandler())

	events := s.Routes.Group("/events")
	events.Post("/flux", s.FluxEventHandler())
	events.Get("/stream", s.EventsStreamHandler())

	configs := s.Routes.Group("/config")
	configs.Get("/", s.GetConfigsHandler())
	configs.Patch("/", s.UpdateConfigsHandler())

	debug := s.Routes.Group("/debug")
	debug.Get("/listeners", s.GetDebugListenersHandler())

	files := s.Routes.Group("/uploads")
	files.Post("/", s.UploadFiles())

	repositories := s.Routes.Group("/repositories")
	repositories.Post("/", s.ValidateRepositoryHandler())

	// mount everything on results
	// TODO it should be named /api/ + dashboard refactor
	s.Mux.Mount("/results", s.Mux)

	// mount dashboard on /ui
	dashboardURI := os.Getenv("TESTKUBE_DASHBOARD_URI")
	if dashboardURI == "" {
		dashboardURI = "http://testkube-dashboard"
	}
	s.Log.Infow("dashboard uri", "uri", dashboardURI)
	s.Mux.All("/", proxy.Forward(dashboardURI))

}

func (s TestkubeAPI) StartTelemetryHeartbeats(ctx context.Context) {

	go func() {
		ticker := time.NewTicker(HeartbeatInterval)
		for {
			telemetryEnabled, err := s.ConfigMap.GetTelemetryEnabled(ctx)
			if err != nil {
				s.Log.Errorw("error getting config map", "error", err)
			}
			if telemetryEnabled {
				l := s.Log.With("measurmentId", telemetry.TestkubeMeasurementID, "secret", text.Obfuscate(telemetry.TestkubeMeasurementSecret))
				host, err := os.Hostname()
				if err != nil {
					l.Debugw("getting hostname error", "hostname", host, "error", err)
				}
				out, err := telemetry.SendHeartbeatEvent(host, version.Version, s.Config.ClusterID)
				if err != nil {
					l.Debugw("sending heartbeat telemetry event error", "error", err)
				} else {
					l.Debugw("sending heartbeat telemetry event", "output", out)
				}

			}
			<-ticker.C
		}
	}()
}

// TODO should we use single generic filter for all list based resources ?
// currently filters for e.g. tests are done "by hand"
func getFilterFromRequest(c *fiber.Ctx) result.Filter {

	filter := result.NewExecutionsFilter()

	// id for /tests/ID/executions
	testName := c.Params("id", "")
	if testName == "" {
		// query param for /executions?testName
		testName = c.Query("testName", "")
	}

	if testName != "" {
		filter = filter.WithTestName(testName)
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
		filter = filter.WithStatus(status)
	}

	objectType := c.Query("type", "")
	if objectType != "" {
		filter = filter.WithType(objectType)
	}

	last, err := strconv.Atoi(c.Query("last", "0"))
	if err == nil && last != 0 {
		filter = filter.WithLastNDays(last)
	}

	dFilter := datefilter.NewDateFilter(c.Query("startDate", ""), c.Query("endDate", ""))
	if dFilter.IsStartValid {
		filter = filter.WithStartDate(dFilter.Start)
	}

	if dFilter.IsEndValid {
		filter = filter.WithEndDate(dFilter.End)
	}

	selector := c.Query("selector")
	if selector != "" {
		filter = filter.WithSelector(selector)
	}

	return filter
}

func (s TestkubeAPI) getSlackLoader() *slack.SlackLoader {
	messageTemplate := os.Getenv("SLACK_TEMPLATE")

	templateDecoded, err := base64.StdEncoding.DecodeString(messageTemplate)
	if err != nil {
		s.Log.Errorw("error while decoding slack template", "error", err.Error())
	}

	configString := os.Getenv("SLACK_CONFIG")
	configDecoded, err := base64.StdEncoding.DecodeString(configString)
	if err != nil {
		s.Log.Errorw("error while decoding slack config", "error", err.Error())
	}

	return slack.NewSlackLoader(string(templateDecoded), string(configDecoded), testkube.AllEventTypes)
}
