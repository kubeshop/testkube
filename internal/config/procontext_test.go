package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestShouldPushClusterInventory(t *testing.T) {
	listener := []cloud.AgentCapability{
		cloud.AgentCapability_AGENT_CAPABILITY_RUNNER,
		cloud.AgentCapability_AGENT_CAPABILITY_EXECUTION,
		cloud.AgentCapability_AGENT_CAPABILITY_LISTENER,
	}
	executionOnly := []cloud.AgentCapability{
		cloud.AgentCapability_AGENT_CAPABILITY_RUNNER,
		cloud.AgentCapability_AGENT_CAPABILITY_EXECUTION,
	}
	tests := []struct {
		name       string
		proContext ProContext
		want       bool
	}{
		{
			name:       "standalone mode never pushes",
			proContext: ProContext{Agent: ProContextAgent{Capabilities: listener}},
			want:       false,
		},
		{
			name:       "connected listener-capable agent pushes",
			proContext: ProContext{APIKey: "key", Agent: ProContextAgent{Capabilities: listener}},
			want:       true,
		},
		{
			name:       "connected execution-only agent does not push",
			proContext: ProContext{APIKey: "key", Agent: ProContextAgent{Capabilities: executionOnly}},
			want:       false,
		},
		{
			name:       "connected agent without capabilities does not push",
			proContext: ProContext{APIKey: "key"},
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldPushClusterInventory(tt.proContext))
		})
	}
}
