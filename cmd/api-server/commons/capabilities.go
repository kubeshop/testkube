package commons

import (
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
)

var capabilityNames = map[cloud.AgentCapability]config.AgentCapability{
	cloud.AgentCapability_AGENT_CAPABILITY_RUNNER:         config.AgentCapabilityRunner,
	cloud.AgentCapability_AGENT_CAPABILITY_LISTENER:       config.AgentCapabilityListener,
	cloud.AgentCapability_AGENT_CAPABILITY_GITOPS:         config.AgentCapabilityGitOps,
	cloud.AgentCapability_AGENT_CAPABILITY_WEBHOOKS:       config.AgentCapabilityWebhooks,
	cloud.AgentCapability_AGENT_CAPABILITY_CLOUD_WEBHOOKS: config.AgentCapabilityCloudWebhooks,
}

// AgentCapabilityNames converts the enum-typed list returned by
// UpdateAgentCapabilitiesOnStartup into the string form ProContextAgent carries.
// Unmapped values are dropped rather than surfaced, matching what the Control
// Plane persists on that path.
func AgentCapabilityNames(capabilities []cloud.AgentCapability) []string {
	names := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		if name, ok := capabilityNames[capability]; ok {
			names = append(names, string(name))
		}
	}
	return names
}
