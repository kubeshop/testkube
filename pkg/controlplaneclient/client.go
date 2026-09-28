package controlplaneclient

import (
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

const (
	AgentSuperAgentPrefix = "tkcagnt"
	AgentRunnerPrefix     = "tkcrun"
)

var _ Client = &client{}

type client struct {
	client     cloud.TestKubeCloudAPIClient
	proContext config.ProContext
	opts       ClientOptions
	logger     *zap.SugaredLogger

	// Persistent notification stream session managers, one per stream kind, created
	// lazily on first use with the agent's long-lived context. Keeping them across gRPC
	// reconnects preserves each session's replay buffer and pod-log source, so a
	// reconnect resumes from the cursor instead of falling back to resume_unavailable.
	notifMu              sync.Mutex
	workflowNotifManager *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest]
	parallelNotifManager *notificationStreamSessionManager[*cloud.TestWorkflowParallelStepNotificationsRequest]
	serviceNotifManager  *notificationStreamSessionManager[*cloud.TestWorkflowServiceNotificationsRequest]
}

type ClientOptions struct {
	StorageSkipVerify  bool
	ExecutionID        string
	WorkflowName       string
	ParentExecutionIDs []string

	// ParentActorType is the actor type of the parent execution. It feeds
	// ChildRunningContextType on the actor type itself, which decides whether
	// the chained child inherits the parent's actor or falls back to the
	// default RunningContextType_EXECUTION (mapped to actor.type = testworkflow).
	ParentActorType testkube.TestWorkflowRunningContextActorType

	Runtime     RuntimeConfig
	SendTimeout time.Duration
	RecvTimeout time.Duration
}

type RuntimeConfig struct {
	Namespace string
}

//go:generate go tool mockgen -destination=./mock_client.go -package=controlplaneclient "github.com/kubeshop/testkube/pkg/controlplaneclient" Client
type Client interface {
	IsSuperAgent() bool
	IsRunner() bool

	ExecutionClient
	ExecutionSelfClient
	ExecutionCacheClient
	RunnerClient
	TestWorkflowsClient
	TestWorkflowTemplatesClient
	TestTriggersClient
	WorkflowTriggersClient
	WebhooksClient
}

func New(grpcClient cloud.TestKubeCloudAPIClient, proContext config.ProContext, opts ClientOptions, logger *zap.SugaredLogger) Client {
	return &client{
		client:     grpcClient,
		proContext: proContext,
		opts:       opts,
		logger:     logger,
	}
}

func (c *client) IsSuperAgent() bool {
	return strings.HasPrefix(c.proContext.APIKey, AgentSuperAgentPrefix+"_")
}

func (c *client) IsRunner() bool {
	return strings.HasPrefix(c.proContext.APIKey, AgentRunnerPrefix+"_")
}
