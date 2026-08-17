package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldPushClusterInventory(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		proContext ProContext
		want       bool
	}{
		{
			name: "standalone mode never pushes",
			cfg:  Config{EnableK8sControllers: true},
			want: false,
		},
		{
			name:       "connected listener-capable agent pushes",
			proContext: ProContext{APIKey: "key", Agent: ProContextAgent{Capabilities: []string{"runner", "listener"}}},
			want:       true,
		},
		{
			name:       "connected runner-only agent does not push, even with the flags on",
			cfg:        Config{EnableK8sControllers: true},
			proContext: ProContext{APIKey: "key", Agent: ProContextAgent{Capabilities: []string{"runner"}}},
			want:       false,
		},
		{
			name:       "unknown capabilities fall back to controllers flag",
			cfg:        Config{EnableK8sControllers: true},
			proContext: ProContext{APIKey: "key"},
			want:       true,
		},
		{
			name:       "unknown capabilities fall back to gitops flag",
			cfg:        Config{GitOpsSyncConfig: GitOpsSyncConfig{GitOpsSyncKubernetesToCloudEnabled: true}},
			proContext: ProContext{APIKey: "key"},
			want:       true,
		},
		{
			name:       "unknown capabilities without listener flags do not push",
			proContext: ProContext{APIKey: "key"},
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldPushClusterInventory(&tt.cfg, tt.proContext))
		})
	}
}
