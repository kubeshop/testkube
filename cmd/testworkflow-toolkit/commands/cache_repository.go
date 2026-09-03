package commands

import (
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/executioncache"
)

// newCacheRepository decides how, or whether, this pod can reach the cache.
//
// The capability is checked here rather than in the processor on purpose. The control
// plane may be upgraded between the moment a workflow is bundled and the moment it runs,
// the pod is the only place that observes the live capability set, and skipping the stage
// at bundle time would also swallow the line that tells an operator why nothing was
// cached.
//
// TODO: return a gRPC-backed repository once the agent-side wiring is implemented
// (calling GetExecutionCachePresigned / SaveExecutionCachePresigned and performing the
// HTTP transfers via the returned presigned URLs).
func newCacheRepository() executioncache.Repository {
	if !capabilities.Enabled(env.GetCapabilities(), capabilities.CapabilityDependencyCache) {
		return executioncache.Unsupported(
			"this control plane does not advertise the \"" + string(capabilities.CapabilityDependencyCache) +
				"\" capability, so nothing is cached")
	}
	return executioncache.Unsupported(
		"the agent in this build cannot transfer dependency caches yet, so nothing is cached")
}
