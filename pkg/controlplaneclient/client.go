package controlplaneclient

import (
	"strings"
	"time"

	"github.com/kubeshop/testkube/internal/config"
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
}

type ClientOptions struct {
	StorageSkipVerify  bool
	ExecutionID        string
	WorkflowName       string
	ParentExecutionIDs []string

	Runtime     RuntimeConfig
	SendTimeout time.Duration
	RecvTimeout time.Duration
}

type RuntimeConfig struct {
	Namespace string
}

//go:generate mockgen -destination=./mock_client.go -package=controlplaneclient "github.com/kubeshop/testkube/pkg/controlplaneclient" Client
type Client interface {
	IsSuperAgent() bool
	IsRunner() bool
	IsLegacy() bool

	ExecutionClient
	ExecutionSelfClient
	RunnerClient
	TestWorkflowsClient
	TestWorkflowTemplatesClient
}

func New(grpcClient cloud.TestKubeCloudAPIClient, proContext config.ProContext, opts ClientOptions) Client {
	return &client{
		client:     grpcClient,
		proContext: proContext,
		opts:       opts,
	}
}

func (c *client) IsSuperAgent() bool {
	return strings.HasPrefix(c.proContext.APIKey, AgentSuperAgentPrefix+"_")
}

func (c *client) IsRunner() bool {
	return strings.HasPrefix(c.proContext.APIKey, AgentRunnerPrefix+"_")
}

func (c *client) IsLegacy() bool {
	return c.IsSuperAgent() && !c.proContext.NewArchitecture
}
