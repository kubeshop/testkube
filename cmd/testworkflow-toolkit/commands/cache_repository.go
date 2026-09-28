package commands

import (
	"context"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/executioncache"
)

// controlPlaneCacheRepository asks the Control Plane where a cache entry lives.
//
// It carries the environment and execution ids rather than taking them per call, because
// they identify this pod for the whole of its life and the Control Plane resolves the
// cache scope from the execution rather than from anything the caller supplies.
type controlPlaneCacheRepository struct {
	client        controlplaneclient.ExecutionCacheClient
	environmentID string
	executionID   string
}

func (r controlPlaneCacheRepository) Restore(ctx context.Context, req executioncache.RestoreRequest) (executioncache.RestoreResult, error) {
	return r.client.GetExecutionCachePresignedURL(ctx, r.environmentID, r.executionID, req)
}

func (r controlPlaneCacheRepository) Save(ctx context.Context, req executioncache.SaveRequest) (executioncache.SaveResult, error) {
	return r.client.SaveExecutionCacheGetPresignedURL(ctx, r.environmentID, r.executionID, req)
}

// newCacheRepository decides how, or whether, this pod can reach the cache.
//
// The capability is checked here rather than in the processor on purpose. The Control
// Plane may be upgraded between the moment a workflow is bundled and the moment it runs,
// the pod is the only place that observes the live capability set, and skipping the stage
// at bundle time would also swallow the line that tells an operator why nothing was
// cached.
//
// Every failure to build a client is an unsupported repository rather than an error: a
// cache is an optimization, so a pod that cannot reach a Control Plane installs from the
// network exactly as it would have without one.
func newCacheRepository() executioncache.Repository {
	if !capabilities.Enabled(env.GetCapabilities(), capabilities.CapabilityDependencyCache) {
		return executioncache.Unsupported(
			"this control plane does not advertise the \"" + string(capabilities.CapabilityDependencyCache) +
				"\" capability, so nothing is cached")
	}

	cfg := config.Config()
	if cfg.Execution.Id == "" {
		return executioncache.Unsupported("this step has no execution id, so nothing is cached")
	}

	client, err := env.Cloud()
	if err != nil {
		return executioncache.Unsupported("cannot reach the control plane (" + err.Error() + "), so nothing is cached")
	}

	return controlPlaneCacheRepository{
		client:        client,
		environmentID: cfg.Execution.EnvironmentId,
		executionID:   cfg.Execution.Id,
	}
}
