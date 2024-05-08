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

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	repoConfig "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"

	"github.com/kubeshop/testkube/pkg/version"

	"github.com/kubeshop/testkube/pkg/datefilter"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"

	"k8s.io/client-go/kubernetes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/kelseyhightower/envconfig"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	templatesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsourcesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testsources/v1"
	testsuitesclientv3 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v3"
	testkubeclientset "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/cdevent"
	"github.com/kubeshop/testkube/pkg/event/kind/slack"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
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
	configMap repoConfig.Repository,
	clusterId string,
	eventsEmitter *event.Emitter,
	executor client.Executor,
	containerExecutor client.Executor,
	metrics metrics.Metrics,
	scheduler *scheduler.Scheduler,
	slackLoader *slack.SlackLoader,
	storage storage.Client,
	graphqlPort string,
	artifactsStorage storage.ArtifactsStorage,
	templatesClient *templatesclientv1.TemplatesClient,
	cdeventsTarget string,
	dashboardURI string,
	helmchartVersion string,
	mode string,
	eventsBus bus.Bus,
	enableSecretsEndpoint bool,
	ff featureflags.FeatureFlags,
	logsStream logsclient.Stream,
	logGrpcClient logsclient.StreamGetter,
	disableSecretCreation bool,
	subscriptionChecker checktcl.SubscriptionChecker,
	serviceAccountNames map[string]string,
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
		HTTPServer:            server.NewServer(httpConfig),
		TestExecutionResults:  testsuiteExecutionsResults,
		ExecutionResults:      testExecutionResults,
		TestsClient:           testsClient,
		ExecutorsClient:       executorsClient,
		SecretClient:          secretClient,
		Clientset:             clientset,
		TestsSuitesClient:     testsuitesClient,
		TestKubeClientset:     testkubeClientset,
		Metrics:               metrics,
		Events:                eventsEmitter,
		WebhooksClient:        webhookClient,
		TestSourcesClient:     testsourcesClient,
		Namespace:             namespace,
		ConfigMap:             configMap,
		Executor:              executor,
		ContainerExecutor:     containerExecutor,
		scheduler:             scheduler,
		slackLoader:           slackLoader,
		Storage:               storage,
		graphqlPort:           graphqlPort,
		ArtifactsStorage:      artifactsStorage,
		TemplatesClient:       templatesClient,
		dashboardURI:          dashboardURI,
		helmchartVersion:      helmchartVersion,
		mode:                  mode,
		eventsBus:             eventsBus,
		enableSecretsEndpoint: enableSecretsEndpoint,
		featureFlags:          ff,
		logsStream:            logsStream,
		logGrpcClient:         logGrpcClient,
		disableSecretCreation: disableSecretCreation,
		SubscriptionChecker:   subscriptionChecker,
		LabelSources:          common.Ptr(make([]LabelSource, 0)),
		ServiceAccountNames:   serviceAccountNames,
	}

	// will be reused in websockets handler
	s.WebsocketLoader = ws.NewWebsocketLoader()

	s.Events.Loader.Register(webhook.NewWebhookLoader(s.Log, webhookClient, templatesClient))
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
	ExecutionResults      result.Repository
	TestExecutionResults  testresult.Repository
	Executor              client.Executor
	ContainerExecutor     client.Executor
	TestsSuitesClient     *testsuitesclientv3.TestSuitesClient
	TestsClient           *testsclientv3.TestsClient
	ExecutorsClient       *executorsclientv1.ExecutorsClient
	SecretClient          *secret.Client
	WebhooksClient        *executorsclientv1.WebhooksClient
	TestKubeClientset     testkubeclientset.Interface
	TestSourcesClient     *testsourcesclientv1.TestSourcesClient
	Metrics               metrics.Metrics
	Storage               storage.Client
	storageParams         storageParams
	Namespace             string
	oauthParams           oauthParams
	WebsocketLoader       *ws.WebsocketLoader
	Events                *event.Emitter
	ConfigMap             repoConfig.Repository
	scheduler             *scheduler.Scheduler
	Clientset             kubernetes.Interface
	slackLoader           *slack.SlackLoader
	graphqlPort           string
	ArtifactsStorage      storage.ArtifactsStorage
	TemplatesClient       *templatesclientv1.TemplatesClient
	dashboardURI          string
	helmchartVersion      string
	mode                  string
	eventsBus             bus.Bus
	enableSecretsEndpoint bool
	featureFlags          featureflags.FeatureFlags
	logsStream            logsclient.Stream
	logGrpcClient         logsclient.StreamGetter
	proContext            *config.ProContext
	disableSecretCreation bool
	SubscriptionChecker   checktcl.SubscriptionChecker
	LabelSources          *[]LabelSource
	ServiceAccountNames   map[string]string
}

type storageParams struct {
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

type oauthParams struct {
	ClientID     string
	ClientSecret string
	Provider     oauth.ProviderType
	Scopes       string
}

func (s *TestkubeAPI) WithFeatureFlags(ff featureflags.FeatureFlags) *TestkubeAPI {
	s.featureFlags = ff
	return s
}

type LabelSource interface {
	ListLabels() (map[string][]string, error)
}

func (s *TestkubeAPI) WithLabelSources(l ...LabelSource) {
	*s.LabelSources = append(*s.LabelSources, l...)
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

	if s.enableSecretsEndpoint {
		files := root.Group("/secrets")
		files.Get("/", s.ListSecretsHandler())
	}

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

// WithProContext sets pro context for the API
func (s *TestkubeAPI) WithProContext(proContext *config.ProContext) *TestkubeAPI {
	s.proContext = proContext
	return s
}

// WithSubscriptionChecker sets subscription checker for the API
// This is used to check if Pro/Enterprise subscription is valid
func (s *TestkubeAPI) WithSubscriptionChecker(subscriptionChecker checktcl.SubscriptionChecker) *TestkubeAPI {
	s.SubscriptionChecker = subscriptionChecker
	return s
}
