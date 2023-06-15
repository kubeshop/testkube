package v1

import (
	"context"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/config"

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
	testsuitesclientv3 "github.com/kubeshop/testkube-operator/client/testsuites/v3"
	testkubeclientset "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/kind/cdevent"
	"github.com/kubeshop/testkube/pkg/event/kind/slack"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/oauth"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

const (
	HeartbeatInterval    = time.Hour
	DefaultHttpBodyLimit = 1 * 1024 * 1024 * 1024 // 1GB - needed for file uploads
)

func NewTestkubeAPI(
	namespace string,
	testExecutionResults result.Repository,
	testsuiteExecutionsResults testresult.Repository,
	testsClient *testsclientv3.TestsClient,
	executorsClient *executorsclientv1.ExecutorsClient,
	testsuitesClient *testsuitesclientv3.TestSuitesClient,
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
	jobTemplate string,
	scheduler *scheduler.Scheduler,
	slackLoader *slack.SlackLoader,
	storage storage.Client,
	graphqlPort string,
	artifactsStorage storage.ArtifactsStorage,
	cdeventsTarget string,
	dashboardURI string,
) TestkubeAPI {

	var httpConfig server.Config
	err := envconfig.Process("APISERVER", &httpConfig)
	// Do we want to panic here or just ignore the error
	if err != nil {
		panic(err)
	}

	httpConfig.ClusterID = clusterId
	httpConfig.Http.BodyLimit = httpConfig.HttpBodyLimit
	if httpConfig.HttpBodyLimit == 0 {
		httpConfig.Http.BodyLimit = DefaultHttpBodyLimit
	}

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
		jobTemplate:          jobTemplate,
		scheduler:            scheduler,
		slackLoader:          slackLoader,
		Storage:              storage,
		graphqlPort:          graphqlPort,
		artifactsStorage:     artifactsStorage,
	}

	// will be reused in websockets handler
	s.WebsocketLoader = ws.NewWebsocketLoader()

	s.Events.Loader.Register(webhook.NewWebhookLoader(webhookClient))
	s.Events.Loader.Register(s.WebsocketLoader)
	s.Events.Loader.Register(s.slackLoader)

	if cdeventsTarget != "" {
		cdeventLoader, err := cdevent.NewCDEventLoader(cdeventsTarget, clusterId, namespace, dashboardURI, testkube.AllEventTypes)
		if err == nil {
			s.Events.Loader.Register(cdeventLoader)
		} else {
			s.Log.Debug("cdevents init error", "error", err.Error())
		}
	}

	s.InitEnvs()
	s.InitRoutes()

	return s
}

type TestkubeAPI struct {
	server.HTTPServer
	ExecutionResults     result.Repository
	TestExecutionResults testresult.Repository
	Executor             client.Executor
	ContainerExecutor    client.Executor
	TestsSuitesClient    *testsuitesclientv3.TestSuitesClient
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
	jobTemplate          string
	scheduler            *scheduler.Scheduler
	Clientset            kubernetes.Interface
	slackLoader          *slack.SlackLoader
	graphqlPort          string
	artifactsStorage     storage.ArtifactsStorage
}

type storageParams struct {
	SSL             bool
	Endpoint        string
	AccessKeyId     string
	SecretAccessKey string
	Region          string
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

// InitEnvs initializes api server settings
func (s *TestkubeAPI) InitEnvs() {
	if err := envconfig.Process("STORAGE", &s.storageParams); err != nil {
		s.Log.Infow("Processing STORAGE environment config", err)
	}

	if err := envconfig.Process("TESTKUBE_OAUTH", &s.oauthParams); err != nil {
		s.Log.Infow("Processing TESTKUBE_OAUTH environment config", err)
	}
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
	executions.Get("/:executionID/artifact-archive", s.GetArtifactArchiveHandler())

	tests := s.Routes.Group("/tests")

	tests.Get("/", s.ListTestsHandler())
	tests.Post("/", s.CreateTestHandler())
	tests.Patch("/:id", s.UpdateTestHandler())
	tests.Delete("/", s.DeleteTestsHandler())

	tests.Get("/:id", s.GetTestHandler())
	tests.Delete("/:id", s.DeleteTestHandler())
	tests.Post("/:id/abort", s.AbortTestHandler())

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
	testsuites.Post("/:id/abort", s.AbortTestSuiteHandler())

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

	// set up proxy for the internal GraphQL server
	s.Mux.All("/graphql", func(c *fiber.Ctx) error {
		// Connect to server
		serverConn, err := net.Dial("tcp", ":"+s.graphqlPort)
		if err != nil {
			s.Log.Errorw("could not connect to GraphQL server as a proxy", "error", err)
			return err
		}

		// Resend headers to the server
		_, err = serverConn.Write(c.Request().Header.Header())
		if err != nil {
			serverConn.Close()
			s.Log.Errorw("error while sending headers to GraphQL server", "error", err)
			return err
		}

		// Resend body to the server
		_, err = serverConn.Write(c.Body())
		if err != nil && err != io.EOF {
			serverConn.Close()
			s.Log.Errorw("error while reading GraphQL client data", "error", err)
			return err
		}

		// Handle optional WebSocket connection
		c.Context().HijackSetNoResponse(true)
		c.Context().Hijack(func(clientConn net.Conn) {
			// Close the connection afterward
			defer serverConn.Close()
			defer clientConn.Close()

			// Extract Unix connection
			serverSock, ok := serverConn.(*net.TCPConn)
			if !ok {
				s.Log.Errorw("error while building TCPConn out ouf serverConn", "error", err)
				return
			}
			clientSock, ok := reflect.Indirect(reflect.ValueOf(clientConn)).FieldByName("Conn").Interface().(*net.TCPConn)
			if !ok {
				s.Log.Errorw("error while building TCPConn out of hijacked connection", "error", err)
				return
			}

			// Duplex communication between client and GraphQL server
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				_, err := io.Copy(clientSock, serverSock)
				if err != nil && err != io.EOF && !errors.Is(err, syscall.ECONNRESET) && !errors.Is(err, syscall.EPIPE) {
					s.Log.Errorw("error while reading GraphQL client data", "error", err)
				}
				serverSock.CloseWrite()
			}()
			go func() {
				defer wg.Done()
				_, err = io.Copy(serverSock, clientSock)
				if err != nil && err != io.EOF {
					s.Log.Errorw("error while reading GraphQL server data", "error", err)
				}
				clientSock.CloseWrite()
			}()
			wg.Wait()
		})
		return nil
	})
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
