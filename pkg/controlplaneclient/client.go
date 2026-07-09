package controlplaneclient

import (
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
)

const (
	AgentSuperAgentPrefix = "tkcagnt"
	AgentRunnerPrefix     = "tkcrun"
)

var _ Client = &client{}

type client struct {
	client            cloud.TestKubeCloudAPIClient
	proContext        config.ProContext
	opts              ClientOptions
	liveLogReplayBudg *liveLogReplayBudget
	logger            *zap.SugaredLogger
}

type ClientOptions struct {
	StorageSkipVerify  bool
	ExecutionID        string
	WorkflowName       string
	ParentExecutionIDs []string

	Runtime     RuntimeConfig
	SendTimeout time.Duration
	RecvTimeout time.Duration

	// LiveLogReplayMaxBytes is the aggregate memory budget shared by all live-log
	// replay buffers on this client. Operator-tunable; when zero, New defaults it
	// to defaultLiveLogReplayMaxBytes.
	LiveLogReplayMaxBytes int64
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
	RunnerClient
	TestWorkflowsClient
	TestWorkflowTemplatesClient
	TestTriggersClient
	WorkflowTriggersClient
	WebhooksClient
}

func New(grpcClient cloud.TestKubeCloudAPIClient, proContext config.ProContext, opts ClientOptions, logger *zap.SugaredLogger) Client {
	return &client{
		client:            grpcClient,
		proContext:        proContext,
		opts:              opts,
		liveLogReplayBudg: newLiveLogReplayBudget(opts.LiveLogReplayMaxBytes),
		logger:            logger,
	}
}

func (c *client) IsSuperAgent() bool {
	return strings.HasPrefix(c.proContext.APIKey, AgentSuperAgentPrefix+"_")
}

func (c *client) IsRunner() bool {
	return strings.HasPrefix(c.proContext.APIKey, AgentRunnerPrefix+"_")
}
