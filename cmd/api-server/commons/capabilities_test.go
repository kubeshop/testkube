package commons

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestAgentCapabilityNames(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []cloud.AgentCapability
		want         []string
	}{
		{
			name:         "empty",
			capabilities: nil,
			want:         []string{},
		},
		{
			name: "every known capability keeps the control plane's spelling",
			capabilities: []cloud.AgentCapability{
				cloud.AgentCapability_AGENT_CAPABILITY_RUNNER,
				cloud.AgentCapability_AGENT_CAPABILITY_LISTENER,
				cloud.AgentCapability_AGENT_CAPABILITY_GITOPS,
				cloud.AgentCapability_AGENT_CAPABILITY_WEBHOOKS,
				cloud.AgentCapability_AGENT_CAPABILITY_CLOUD_WEBHOOKS,
			},
			want: []string{"runner", "listener", "gitops", "webhooks", "cloud-webhooks"},
		},
		{
			name: "unspecified is dropped",
			capabilities: []cloud.AgentCapability{
				cloud.AgentCapability_AGENT_CAPABILITY_UNSPECIFIED,
				cloud.AgentCapability_AGENT_CAPABILITY_RUNNER,
			},
			want: []string{"runner"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AgentCapabilityNames(tt.capabilities))
		})
	}
}
