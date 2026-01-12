package runner

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	runnergrpc "github.com/kubeshop/testkube/pkg/runner/grpc"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

// NewMetrics returns a new metrics instance.
func NewMetrics() metrics.Metrics {
	return metrics.NewMetrics()
}

// NewControlPlaneClient creates a control plane client using the provided credentials.
func NewControlPlaneClient(
	grpcClient cloud.TestKubeCloudAPIClient,
	credentials AgentCredentials,
	namespace string,
	logger *zap.SugaredLogger,
) controlplaneclient.Client {
	proContext := config.ProContext{
		APIKey:      credentials.APIKey,
		URL:         credentials.URL,
		OrgID:       credentials.OrgID,
		EnvID:       credentials.EnvID,
		SkipVerify:  credentials.SkipVerify,
		TLSInsecure: credentials.TLSInsecure,
	}
	return controlplaneclient.New(grpcClient, proContext, controlplaneclient.ClientOptions{
		StorageSkipVerify: credentials.SkipVerify,
		Runtime:           controlplaneclient.RuntimeConfig{Namespace: namespace},
		SendTimeout:       30 * time.Second,
		RecvTimeout:       30 * time.Second,
	}, logger)
}

// AgentIdentity contains agent identity information.
type AgentIdentity struct {
	ID     string
	Name   string
	Labels map[string]string
}

// AgentCredentials contains the credentials needed to authenticate with the control plane.
type AgentCredentials struct {
	APIKey      string
	OrgID       string
	EnvID       string
	URL         string
	SkipVerify  bool
	TLSInsecure bool
}

// AgentConfig contains all dependencies needed to create an agent.
type AgentConfig struct {
	// Core dependencies
	ExecutionWorker    executionworkertypes.Worker
	ConfigRepository   configRepo.Repository
	ControlPlaneClient controlplaneclient.Client
	EventsEmitter      *event.Emitter
	Metrics            metrics.Metrics
	Logger             *zap.SugaredLogger

	// Agent identity and credentials (two options):
	// Option 1: Use ProContext directly (for main.go and internal packages)
	ProContext config.ProContext
	// Option 2: Use public types (for external packages like control plane tests)
	Agent       AgentIdentity
	Credentials AgentCredentials

	// Runner options
	Options Options

	// Control plane config
	ControlPlaneConfig testworkflowconfig.ControlPlaneConfig

	// gRPC connection for runnerClient
	GRPCConn            *grpc.ClientConn
	GRPCTLSEnabled      bool
	TestWorkflowsClient testworkflowclient.TestWorkflowClient
}

// buildProContext returns the ProContext, either from the direct field
// or by building from the public AgentCredentials/AgentIdentity types.
func (cfg *AgentConfig) buildProContext() config.ProContext {
	// If ProContext is set directly, use it (for main.go and internal packages)
	if cfg.ProContext.APIKey != "" {
		return cfg.ProContext
	}
	// Otherwise build from public types (for external packages)
	return config.ProContext{
		APIKey:      cfg.Credentials.APIKey,
		URL:         cfg.Credentials.URL,
		OrgID:       cfg.Credentials.OrgID,
		EnvID:       cfg.Credentials.EnvID,
		SkipVerify:  cfg.Credentials.SkipVerify,
		TLSInsecure: cfg.Credentials.TLSInsecure,
		Agent: config.ProContextAgent{
			ID:     cfg.Agent.ID,
			Name:   cfg.Agent.Name,
			Labels: cfg.Agent.Labels,
		},
	}
}

// Agent represents a testkube agent with all its components.
// Use Start() to run the agent, or access components directly for custom startup.
type Agent struct {
	Runner       Runner
	Service      Service
	RunnerClient runnergrpc.Client
	ProContext   config.ProContext
}

// NewAgent creates a testkube agent with all its components but does not start it.
// The caller controls startup via the returned Agent's components or Start() method.
func NewAgent(cfg AgentConfig) (*Agent, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}

	if cfg.GRPCConn == nil {
		return nil, fmt.Errorf("GRPCConn is required")
	}

	proContext := cfg.buildProContext()

	runner := New(
		cfg.ExecutionWorker,
		cfg.ConfigRepository,
		cfg.ControlPlaneClient,
		cfg.EventsEmitter,
		cfg.Metrics,
		proContext,
		cfg.Options.StorageSkipVerify,
		cfg.Options.GlobalTemplate,
	)

	service := NewService(
		logger,
		cfg.EventsEmitter,
		cfg.ControlPlaneClient,
		cfg.ControlPlaneConfig,
		proContext,
		cfg.ExecutionWorker,
		cfg.Options,
		runner,
	)

	runnerClient := runnergrpc.NewClient(
		cfg.GRPCConn,
		logger,
		runner,
		proContext.APIKey,
		proContext.OrgID,
		cfg.GRPCTLSEnabled,
		cfg.ControlPlaneConfig,
		cfg.TestWorkflowsClient,
	)

	return &Agent{
		Runner:       runner,
		Service:      service,
		RunnerClient: runnerClient,
		ProContext:   proContext,
	}, nil
}

// Start runs the agent's service and runnerClient concurrently.
// This blocks until ctx is cancelled or an error occurs.
func (a *Agent) Start(ctx context.Context) error {
	var g errgroup.Group
	g.Go(func() error {
		return a.Service.Start(ctx, false)
	})
	g.Go(func() error {
		return a.RunnerClient.Start(ctx, a.ProContext.EnvID)
	})
	return g.Wait()
}
