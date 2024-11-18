package scheduler

import (
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/repository/config"

	v1 "github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/featureflags"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
)

type Scheduler struct {
	metrics                v1.Metrics
	executor               client.Executor
	containerExecutor      client.Executor
	deprecatedRepositories commons.DeprecatedRepositories
	deprecatedClients      commons.DeprecatedClients
	secretClient           secret.Interface
	events                 *event.Emitter
	logger                 *zap.SugaredLogger
	configMap              config.Repository
	configMapClient        configmap.Interface
	eventsBus              bus.Bus
	dashboardURI           string
	featureFlags           featureflags.FeatureFlags
	logsStream             logsclient.Stream
	subscriptionChecker    checktcl.SubscriptionChecker
	namespace              string
	agentAPITLSSecret      string
	runnerCustomCASecret   string
}

func NewScheduler(
	metrics v1.Metrics,
	executor client.Executor,
	containerExecutor client.Executor,
	deprecatedRepositories commons.DeprecatedRepositories,
	deprecatedClients commons.DeprecatedClients,
	secretClient secret.Interface,
	events *event.Emitter,
	logger *zap.SugaredLogger,
	configMap config.Repository,
	configMapClient configmap.Interface,
	eventsBus bus.Bus,
	dashboardURI string,
	featureFlags featureflags.FeatureFlags,
	logsStream logsclient.Stream,
	namespace string,
	agentAPITLSSecret string,
	runnerCustomCASecret string,
	subscriptionChecker checktcl.SubscriptionChecker,
) *Scheduler {
	return &Scheduler{
		metrics:                metrics,
		executor:               executor,
		containerExecutor:      containerExecutor,
		secretClient:           secretClient,
		deprecatedRepositories: deprecatedRepositories,
		deprecatedClients:      deprecatedClients,
		events:                 events,
		logger:                 logger,
		configMap:              configMap,
		configMapClient:        configMapClient,
		eventsBus:              eventsBus,
		dashboardURI:           dashboardURI,
		featureFlags:           featureFlags,
		logsStream:             logsStream,
		namespace:              namespace,
		agentAPITLSSecret:      agentAPITLSSecret,
		runnerCustomCASecret:   runnerCustomCASecret,
		subscriptionChecker:    subscriptionChecker,
	}
}
