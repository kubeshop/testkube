package v1

import (
	"go.uber.org/zap"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	testtriggersclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	repoConfig "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func NewTestkubeAPI(
	deprecatedClients commons.DeprecatedClients,
	clusterId string,
	namespace string,
	testWorkflowResults testworkflow.Repository,
	testWorkflowOutput testworkflow.OutputRepository,
	artifactsStorage storage.ArtifactsStorage,
	webhookClient executorsclientv1.WebhooksInterface,
	webhookTemplateClient executorsclientv1.WebhookTemplatesInterface,
	testTriggersClient testtriggersclientv1.Interface,
	testWorkflowsClient testworkflowclient.TestWorkflowClient,
	testWorkflowsK8SClient testworkflowsv1.Interface,
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	testWorkflowTemplatesK8SClient testworkflowsv1.TestWorkflowTemplatesInterface,
	configMap repoConfig.Repository,
	secretManager secretmanager.SecretManager,
	secretConfig testkube.SecretConfig,
	executionWorkerClient executionworkertypes.Worker,
	eventsEmitter *event.Emitter,
	websocketLoader *ws.WebsocketLoader,
	metrics metrics.Metrics,
	proContext *config.ProContext,
	ff featureflags.FeatureFlags,
	dashboardURI string,
	helmchartVersion string,
	serviceAccountNames map[string]string,
	dockerImageVersion string,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
) TestkubeAPI {

	return TestkubeAPI{
		ClusterID:                      clusterId,
		Log:                            log.DefaultLogger,
		DeprecatedClients:              deprecatedClients,
		TestWorkflowResults:            testWorkflowResults,
		TestWorkflowOutput:             testWorkflowOutput,
		SecretManager:                  secretManager,
		TestTriggersClient:             testTriggersClient,
		TestWorkflowsClient:            testWorkflowsClient,
		TestWorkflowTemplatesClient:    testWorkflowTemplatesClient,
		TestWorkflowsK8SClient:         testWorkflowsK8SClient,
		TestWorkflowTemplatesK8SClient: testWorkflowTemplatesK8SClient,
		Metrics:                        metrics,
		WebsocketLoader:                websocketLoader,
		Events:                         eventsEmitter,
		WebhooksClient:                 webhookClient,
		WebhookTemplatesClient:         webhookTemplateClient,
		Namespace:                      namespace,
		ConfigMap:                      configMap,
		ExecutionWorkerClient:          executionWorkerClient,
		ArtifactsStorage:               artifactsStorage,
		dashboardURI:                   dashboardURI,
		helmchartVersion:               helmchartVersion,
		secretConfig:                   secretConfig,
		featureFlags:                   ff,
		ServiceAccountNames:            serviceAccountNames,
		dockerImageVersion:             dockerImageVersion,
		proContext:                     proContext,
		testWorkflowExecutor:           testWorkflowExecutor,
	}
}

type TestkubeAPI struct {
	ClusterID                      string
	Log                            *zap.SugaredLogger
	TestWorkflowResults            testworkflow.Repository
	TestWorkflowOutput             testworkflow.OutputRepository
	Executor                       client.Executor
	ContainerExecutor              client.Executor
	ExecutionWorkerClient          executionworkertypes.Worker
	DeprecatedClients              commons.DeprecatedClients
	SecretManager                  secretmanager.SecretManager
	WebhooksClient                 executorsclientv1.WebhooksInterface
	WebhookTemplatesClient         executorsclientv1.WebhookTemplatesInterface
	TestTriggersClient             testtriggersclientv1.Interface
	TestWorkflowsClient            testworkflowclient.TestWorkflowClient
	TestWorkflowTemplatesClient    testworkflowtemplateclient.TestWorkflowTemplateClient
	TestWorkflowsK8SClient         testworkflowsv1.Interface
	TestWorkflowTemplatesK8SClient testworkflowsv1.TestWorkflowTemplatesInterface
	Metrics                        metrics.Metrics
	Namespace                      string
	WebsocketLoader                *ws.WebsocketLoader
	Events                         *event.Emitter
	ConfigMap                      repoConfig.Repository
	ArtifactsStorage               storage.ArtifactsStorage
	dashboardURI                   string
	helmchartVersion               string
	secretConfig                   testkube.SecretConfig
	featureFlags                   featureflags.FeatureFlags
	proContext                     *config.ProContext
	ServiceAccountNames            map[string]string
	dockerImageVersion             string
	testWorkflowExecutor           testworkflowexecutor.TestWorkflowExecutor
}

func (s *TestkubeAPI) Init(server server.HTTPServer) {
	// TODO: Consider extracting outside?
	server.Routes.Get("/info", s.InfoHandler())
	server.Routes.Get("/debug", s.DebugHandler())

	root := server.Routes

	webhooks := root.Group("/webhooks")

	webhooks.Post("/", s.CreateWebhookHandler())
	webhooks.Patch("/:name", s.UpdateWebhookHandler())
	webhooks.Get("/", s.ListWebhooksHandler())
	webhooks.Get("/:name", s.GetWebhookHandler())
	webhooks.Delete("/:name", s.DeleteWebhookHandler())
	webhooks.Delete("/", s.DeleteWebhooksHandler())

	webhookTemplates := root.Group("/webhook-templates")

	webhookTemplates.Post("/", s.CreateWebhookTemplateHandler())
	webhookTemplates.Patch("/:name", s.UpdateWebhookTemplateHandler())
	webhookTemplates.Get("/", s.ListWebhookTemplatesHandler())
	webhookTemplates.Get("/:name", s.GetWebhookTemplateHandler())
	webhookTemplates.Delete("/:name", s.DeleteWebhookTemplateHandler())
	webhookTemplates.Delete("/", s.DeleteWebhookTemplatesHandler())

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
	testWorkflowExecutions.Post("/", s.ExecuteTestWorkflowHandler())
	testWorkflowExecutions.Get("/:executionID", s.GetTestWorkflowExecutionHandler())
	testWorkflowExecutions.Get("/:executionID/notifications", s.StreamTestWorkflowExecutionNotificationsHandler())
	testWorkflowExecutions.Get("/:executionID/notifications/services/:serviceName/:serviceIndex<int>", s.StreamTestWorkflowExecutionServiceNotificationsHandler())
	testWorkflowExecutions.Get("/:executionID/notifications/parallel-steps/:ref/:workerIndex<int>", s.StreamTestWorkflowExecutionParallelStepNotificationsHandler())
	testWorkflowExecutions.Get("/:executionID/notifications/stream", s.StreamTestWorkflowExecutionNotificationsWebSocketHandler())
	testWorkflowExecutions.Get("/:executionID/notifications/stream/services/:serviceName/:serviceIndex<int>", s.StreamTestWorkflowExecutionServiceNotificationsWebSocketHandler())
	testWorkflowExecutions.Get("/:executionID/notifications/stream/parallel-steps/:ref/:workerIndex<int>", s.StreamTestWorkflowExecutionParallelStepNotificationsWebSocketHandler())
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

	labels := root.Group("/labels")
	labels.Get("/", s.ListLabelsHandler())

	tags := root.Group("/tags")
	tags.Get("/", s.ListTagsHandler())

	events := root.Group("/events")
	events.Post("/flux", s.FluxEventHandler())
	events.Get("/stream", s.EventsStreamHandler())

	configs := root.Group("/config")
	configs.Get("/", s.GetConfigsHandler())
	configs.Patch("/", s.UpdateConfigsHandler())

	debug := root.Group("/debug")
	debug.Get("/listeners", s.GetDebugListenersHandler())

	secrets := root.Group("/secrets")
	secrets.Get("/", s.ListSecretsHandler())
	secrets.Post("/", s.CreateSecretHandler())
	secrets.Get("/:id", s.GetSecretHandler())
	secrets.Delete("/:id", s.DeleteSecretHandler())
	secrets.Patch("/:id", s.UpdateSecretHandler())

	repositories := root.Group("/repositories")
	repositories.Post("/", s.ValidateRepositoryHandler())
}
