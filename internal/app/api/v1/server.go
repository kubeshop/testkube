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

	testtriggersclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	repoConfig "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"

	"github.com/kubeshop/testkube/pkg/version"

	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/datefilter"
	"github.com/kubeshop/testkube/pkg/repository/result"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/proxy"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/slack"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/featureflags"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
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
	httpConfig server.Config,
	deprecatedRepositories commons.DeprecatedRepositories,
	deprecatedClients commons.DeprecatedClients,
	namespace string,
	testWorkflowResults testworkflow.Repository,
	testWorkflowOutput testworkflow.OutputRepository,
	secretClient secret.Interface,
	secretManager secretmanager.SecretManager,
	webhookClient *executorsclientv1.WebhooksClient,
	clientset kubernetes.Interface,
	testTriggersClient testtriggersclientv1.Interface,
	testWorkflowsClient testworkflowsv1.Interface,
	testWorkflowTemplatesClient testworkflowsv1.TestWorkflowTemplatesInterface,
	configMap repoConfig.Repository,
	eventsEmitter *event.Emitter,
	websocketLoader *ws.WebsocketLoader,
	executor client.Executor,
	containerExecutor client.Executor,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	executionWorkerClient executionworkertypes.Worker,
	metrics metrics.Metrics,
	scheduler *scheduler.Scheduler,
	slackLoader *slack.SlackLoader,
	graphqlPort string,
	artifactsStorage storage.ArtifactsStorage,
	dashboardURI string,
	helmchartVersion string,
	mode string,
	eventsBus bus.Bus,
	secretConfig testkube.SecretConfig,
	ff featureflags.FeatureFlags,
	logsStream logsclient.Stream,
	logGrpcClient logsclient.StreamGetter,
	serviceAccountNames map[string]string,
	dockerImageVersion string,
	proContext *config.ProContext,
	storageParams StorageParams,
	oauthParams OauthParams,
) TestkubeAPI {

	return TestkubeAPI{
		HTTPServer:                  server.NewServer(httpConfig),
		DeprecatedRepositories:      deprecatedRepositories,
		DeprecatedClients:           deprecatedClients,
		TestWorkflowResults:         testWorkflowResults,
		TestWorkflowOutput:          testWorkflowOutput,
		SecretClient:                secretClient,
		SecretManager:               secretManager,
		Clientset:                   clientset,
		TestTriggersClient:          testTriggersClient,
		TestWorkflowsClient:         testWorkflowsClient,
		TestWorkflowTemplatesClient: testWorkflowTemplatesClient,
		Metrics:                     metrics,
		WebsocketLoader:             websocketLoader,
		Events:                      eventsEmitter,
		WebhooksClient:              webhookClient,
		Namespace:                   namespace,
		ConfigMap:                   configMap,
		Executor:                    executor,
		ContainerExecutor:           containerExecutor,
		TestWorkflowExecutor:        testWorkflowExecutor,
		ExecutionWorkerClient:       executionWorkerClient,
		storageParams:               storageParams,
		oauthParams:                 oauthParams,
		scheduler:                   scheduler,
		slackLoader:                 slackLoader,
		graphqlPort:                 graphqlPort,
		ArtifactsStorage:            artifactsStorage,
		dashboardURI:                dashboardURI,
		helmchartVersion:            helmchartVersion,
		mode:                        mode,
		eventsBus:                   eventsBus,
		secretConfig:                secretConfig,
		featureFlags:                ff,
		logsStream:                  logsStream,
		logGrpcClient:               logGrpcClient,
		ServiceAccountNames:         serviceAccountNames,
		dockerImageVersion:          dockerImageVersion,
		proContext:                  proContext,
	}
}

type TestkubeAPI struct {
	server.HTTPServer
	TestWorkflowResults         testworkflow.Repository
	TestWorkflowOutput          testworkflow.OutputRepository
	Executor                    client.Executor
	ContainerExecutor           client.Executor
	TestWorkflowExecutor        testworkflowexecutor.TestWorkflowExecutor
	ExecutionWorkerClient       executionworkertypes.Worker
	DeprecatedRepositories      commons.DeprecatedRepositories
	DeprecatedClients           commons.DeprecatedClients
	SecretClient                secret.Interface
	SecretManager               secretmanager.SecretManager
	WebhooksClient              *executorsclientv1.WebhooksClient
	TestTriggersClient          testtriggersclientv1.Interface
	TestWorkflowsClient         testworkflowsv1.Interface
	TestWorkflowTemplatesClient testworkflowsv1.TestWorkflowTemplatesInterface
	Metrics                     metrics.Metrics
	storageParams               StorageParams
	Namespace                   string
	oauthParams                 OauthParams
	WebsocketLoader             *ws.WebsocketLoader
	Events                      *event.Emitter
	ConfigMap                   repoConfig.Repository
	scheduler                   *scheduler.Scheduler
	Clientset                   kubernetes.Interface
	slackLoader                 *slack.SlackLoader
	graphqlPort                 string
	ArtifactsStorage            storage.ArtifactsStorage
	dashboardURI                string
	helmchartVersion            string
	mode                        string
	eventsBus                   bus.Bus
	secretConfig                testkube.SecretConfig
	featureFlags                featureflags.FeatureFlags
	logsStream                  logsclient.Stream
	logGrpcClient               logsclient.StreamGetter
	proContext                  *config.ProContext
	ServiceAccountNames         map[string]string
	dockerImageVersion          string
}

type StorageParams struct {
	SSL             bool   `envconfig:"STORAGE_SSL" default:"false"`
	SkipVerify      bool   `envconfig:"STORAGE_SKIP_VERIFY" default:"false"`
	CertFile        string `envconfig:"STORAGE_CERT_FILE"`
	KeyFile         string `envconfig:"STORAGE_KEY_FILE"`
	CAFile          string `envconfig:"STORAGE_CA_FILE"`
	Endpoint        string
	AccessKeyId     string
	SecretAccessKey string
	Region          string
	Token           string
	Bucket          string
}

type OauthParams struct {
	ClientID     string
	ClientSecret string
	Provider     oauth.ProviderType
	Scopes       string
}

// SendTelemetryStartEvent sends anonymous start event to telemetry trackers
func (s TestkubeAPI) SendTelemetryStartEvent(ctx context.Context, ch chan struct{}) {
	go func() {
		defer func() {
			ch <- struct{}{}
		}()

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
	}()
}

func (s *TestkubeAPI) Init() {
	s.InitRoutes()
}

func (s *TestkubeAPI) InitRoutes() {
	s.Routes.Static("/api-docs", "./api/v1")
	s.Routes.Use(cors.New())
	s.Routes.Use(s.AuthHandler())

	s.Routes.Get("/info", s.InfoHandler())
	s.Routes.Get("/routes", s.RoutesHandler())
	s.Routes.Get("/debug", s.DebugHandler())

	root := s.Routes

	executors := root.Group("/executors")

	executors.Post("/", s.CreateExecutorHandler())
	executors.Get("/", s.ListExecutorsHandler())
	executors.Get("/:name", s.GetExecutorHandler())
	executors.Patch("/:name", s.UpdateExecutorHandler())
	executors.Delete("/:name", s.DeleteExecutorHandler())
	executors.Delete("/", s.DeleteExecutorsHandler())

	executorByTypes := root.Group("/executor-by-types")
	executorByTypes.Get("/", s.GetExecutorByTestTypeHandler())

	webhooks := root.Group("/webhooks")

	webhooks.Post("/", s.CreateWebhookHandler())
	webhooks.Patch("/:name", s.UpdateWebhookHandler())
	webhooks.Get("/", s.ListWebhooksHandler())
	webhooks.Get("/:name", s.GetWebhookHandler())
	webhooks.Delete("/:name", s.DeleteWebhookHandler())
	webhooks.Delete("/", s.DeleteWebhooksHandler())

	executions := root.Group("/executions")

	executions.Get("/", s.ListExecutionsHandler())
	executions.Post("/", s.ExecuteTestsHandler())
	executions.Get("/:executionID", s.GetExecutionHandler())
	executions.Get("/:executionID/artifacts", s.ListArtifactsHandler())
	executions.Get("/:executionID/logs", s.ExecutionLogsHandler())
	executions.Get("/:executionID/logs/stream", s.ExecutionLogsStreamHandler())
	executions.Get("/:executionID/logs/v2", s.ExecutionLogsHandlerV2())
	executions.Get("/:executionID/logs/stream/v2", s.ExecutionLogsStreamHandlerV2())
	executions.Get("/:executionID/artifacts/:filename", s.GetArtifactHandler())
	executions.Get("/:executionID/artifact-archive", s.GetArtifactArchiveHandler())

	tests := root.Group("/tests")

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

	testsuites := root.Group("/test-suites")

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

	testSuiteExecutions := root.Group("/test-suite-executions")
	testSuiteExecutions.Get("/", s.ListTestSuiteExecutionsHandler())
	testSuiteExecutions.Post("/", s.ExecuteTestSuitesHandler())
	testSuiteExecutions.Get("/:executionID", s.GetTestSuiteExecutionHandler())
	testSuiteExecutions.Get("/:executionID/artifacts", s.ListTestSuiteArtifactsHandler())
	testSuiteExecutions.Patch("/:executionID", s.AbortTestSuiteExecutionHandler())

	testSuiteWithExecutions := root.Group("/test-suite-with-executions")
	testSuiteWithExecutions.Get("/", s.ListTestSuiteWithExecutionsHandler())
	testSuiteWithExecutions.Get("/:id", s.GetTestSuiteWithExecutionHandler())

	testWorkflows := root.Group("/test-workflows")
	testWorkflows.Get("/", s.ListTestWorkflowsHandler())
	testWorkflows.Post("/", s.CreateTestWorkflowHandler())
	testWorkflows.Delete("/", s.DeleteTestWorkflowsHandler())
	testWorkflows.Get("/:id", s.GetTestWorkflowHandler())
	testWorkflows.Put("/:id", s.UpdateTestWorkflowHandler())
	testWorkflows.Delete("/:id", s.DeleteTestWorkflowHandler())
	testWorkflows.Get("/:id/executions", s.ListTestWorkflowExecutionsHandler())
	testWorkflows.Post("/:id/executions", s.ExecuteTestWorkflowHandler())
	testWorkflows.Get("/:id/tags", s.ListTagsHandler())
	testWorkflows.Get("/:id/metrics", s.GetTestWorkflowMetricsHandler())
	testWorkflows.Get("/:id/executions/:executionID", s.GetTestWorkflowExecutionHandler())
	testWorkflows.Post("/:id/abort", s.AbortAllTestWorkflowExecutionsHandler())
	testWorkflows.Post("/:id/executions/:executionID/abort", s.AbortTestWorkflowExecutionHandler())
	testWorkflows.Post("/:id/executions/:executionID/pause", s.PauseTestWorkflowExecutionHandler())
	testWorkflows.Post("/:id/executions/:executionID/resume", s.ResumeTestWorkflowExecutionHandler())
	testWorkflows.Get("/:id/executions/:executionID/logs", s.GetTestWorkflowExecutionLogsHandler())

	testWorkflowExecutions := root.Group("/test-workflow-executions")
	testWorkflowExecutions.Get("/", s.ListTestWorkflowExecutionsHandler())
	testWorkflowExecutions.Get("/:executionID", s.GetTestWorkflowExecutionHandler())
	testWorkflowExecutions.Get("/:executionID/notifications", s.StreamTestWorkflowExecutionNotificationsHandler())
	testWorkflowExecutions.Get("/:executionID/notifications/stream", s.StreamTestWorkflowExecutionNotificationsWebSocketHandler())
	testWorkflowExecutions.Post("/:executionID/abort", s.AbortTestWorkflowExecutionHandler())
	testWorkflowExecutions.Post("/:executionID/pause", s.PauseTestWorkflowExecutionHandler())
	testWorkflowExecutions.Post("/:executionID/resume", s.ResumeTestWorkflowExecutionHandler())
	testWorkflowExecutions.Get("/:executionID/logs", s.GetTestWorkflowExecutionLogsHandler())
	testWorkflowExecutions.Get("/:executionID/artifacts", s.ListTestWorkflowExecutionArtifactsHandler())
	testWorkflowExecutions.Get("/:executionID/artifacts/:filename", s.GetTestWorkflowArtifactHandler())
	testWorkflowExecutions.Get("/:executionID/artifact-archive", s.GetTestWorkflowArtifactArchiveHandler())

	testWorkflowWithExecutions := root.Group("/test-workflow-with-executions")
	testWorkflowWithExecutions.Get("/", s.ListTestWorkflowWithExecutionsHandler())
	testWorkflowWithExecutions.Get("/:id", s.GetTestWorkflowWithExecutionHandler())
	testWorkflowWithExecutions.Get("/:id/tags", s.ListTagsHandler())

	root.Post("/preview-test-workflow", s.PreviewTestWorkflowHandler())

	testWorkflowTemplates := root.Group("/test-workflow-templates")
	testWorkflowTemplates.Get("/", s.ListTestWorkflowTemplatesHandler())
	testWorkflowTemplates.Post("/", s.CreateTestWorkflowTemplateHandler())
	testWorkflowTemplates.Delete("/", s.DeleteTestWorkflowTemplatesHandler())
	testWorkflowTemplates.Get("/:id", s.GetTestWorkflowTemplateHandler())
	testWorkflowTemplates.Put("/:id", s.UpdateTestWorkflowTemplateHandler())
	testWorkflowTemplates.Delete("/:id", s.DeleteTestWorkflowTemplateHandler())

	testTriggers := root.Group("/triggers")
	testTriggers.Get("/", s.ListTestTriggersHandler())
	testTriggers.Post("/", s.CreateTestTriggerHandler())
	testTriggers.Patch("/", s.BulkUpdateTestTriggersHandler())
	testTriggers.Delete("/", s.DeleteTestTriggersHandler())
	testTriggers.Get("/:id", s.GetTestTriggerHandler())
	testTriggers.Patch("/:id", s.UpdateTestTriggerHandler())
	testTriggers.Delete("/:id", s.DeleteTestTriggerHandler())

	keymap := root.Group("/keymap")
	keymap.Get("/triggers", s.GetTestTriggerKeyMapHandler())

	testsources := root.Group("/test-sources")
	testsources.Post("/", s.CreateTestSourceHandler())
	testsources.Get("/", s.ListTestSourcesHandler())
	testsources.Patch("/", s.ProcessTestSourceBatchHandler())
	testsources.Get("/:name", s.GetTestSourceHandler())
	testsources.Patch("/:name", s.UpdateTestSourceHandler())
	testsources.Delete("/:name", s.DeleteTestSourceHandler())
	testsources.Delete("/", s.DeleteTestSourcesHandler())

	templates := root.Group("/templates")

	templates.Post("/", s.CreateTemplateHandler())
	templates.Patch("/:name", s.UpdateTemplateHandler())
	templates.Get("/", s.ListTemplatesHandler())
	templates.Get("/:name", s.GetTemplateHandler())
	templates.Delete("/:name", s.DeleteTemplateHandler())
	templates.Delete("/", s.DeleteTemplatesHandler())

	labels := root.Group("/labels")
	labels.Get("/", s.ListLabelsHandler())

	tags := root.Group("/tags")
	tags.Get("/", s.ListTagsHandler())

	slack := root.Group("/slack")
	slack.Get("/", s.OauthHandler())

	events := root.Group("/events")
	events.Post("/flux", s.FluxEventHandler())
	events.Get("/stream", s.EventsStreamHandler())

	configs := root.Group("/config")
	configs.Get("/", s.GetConfigsHandler())
	configs.Patch("/", s.UpdateConfigsHandler())

	debug := root.Group("/debug")
	debug.Get("/listeners", s.GetDebugListenersHandler())

	files := root.Group("/uploads")
	files.Post("/", s.UploadFiles())

	secrets := root.Group("/secrets")
	secrets.Get("/", s.ListSecretsHandler())
	secrets.Post("/", s.CreateSecretHandler())
	secrets.Get("/:id", s.GetSecretHandler())
	secrets.Delete("/:id", s.DeleteSecretHandler())
	secrets.Patch("/:id", s.UpdateSecretHandler())

	repositories := root.Group("/repositories")
	repositories.Post("/", s.ValidateRepositoryHandler())

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

func (s TestkubeAPI) StartTelemetryHeartbeats(ctx context.Context, ch chan struct{}) {
	go func() {
		<-ch

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
