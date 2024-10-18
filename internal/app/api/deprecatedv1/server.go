package deprecatedv1

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	testtriggersclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	repoConfig "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"

	"k8s.io/client-go/kubernetes"

	"github.com/gofiber/fiber/v2"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/slack"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/featureflags"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/storage"
)

func NewDeprecatedTestkubeAPI(
	clusterId string,
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
	graphqlPort int,
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
) DeprecatedTestkubeAPI {

	return DeprecatedTestkubeAPI{
		ClusterID:                   clusterId,
		Log:                         log.DefaultLogger,
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

type DeprecatedTestkubeAPI struct {
	ClusterID                   string
	Log                         *zap.SugaredLogger
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
	WebsocketLoader             *ws.WebsocketLoader
	Events                      *event.Emitter
	ConfigMap                   repoConfig.Repository
	scheduler                   *scheduler.Scheduler
	Clientset                   kubernetes.Interface
	slackLoader                 *slack.SlackLoader
	graphqlPort                 int
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

func (s *DeprecatedTestkubeAPI) Init(server server.HTTPServer) {
	root := server.Routes

	executors := root.Group("/executors")

	executors.Post("/", s.CreateExecutorHandler())
	executors.Get("/", s.ListExecutorsHandler())
	executors.Get("/:name", s.GetExecutorHandler())
	executors.Patch("/:name", s.UpdateExecutorHandler())
	executors.Delete("/:name", s.DeleteExecutorHandler())
	executors.Delete("/", s.DeleteExecutorsHandler())

	executorByTypes := root.Group("/executor-by-types")
	executorByTypes.Get("/", s.GetExecutorByTestTypeHandler())

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

	testWithExecutions := server.Routes.Group("/test-with-executions")
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

	slack := root.Group("/slack")
	slack.Get("/", s.OauthHandler())

	files := root.Group("/uploads")
	files.Post("/", s.UploadFiles())

	// set up proxy for the internal GraphQL server
	server.Mux.All("/graphql", func(c *fiber.Ctx) error {
		// Connect to server
		serverConn, err := net.Dial("tcp", fmt.Sprintf(":%d", s.graphqlPort))
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
