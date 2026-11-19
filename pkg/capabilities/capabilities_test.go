package capabilities

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/cloud"
)

func TestEnabled(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []*cloud.Capability
		want         bool
	}{
		{
			name: "reports the multi-namespace sync capability when the control plane enables it",
			capabilities: []*cloud.Capability{
				{Name: string(CapabilitySourceOfTruth), Enabled: true},
				{Name: "sync-multi-namespace", Enabled: true},
			},
			want: true,
		},
		{
			name: "reports false when the capability is listed but disabled",
			capabilities: []*cloud.Capability{
				{Name: "sync-multi-namespace", Enabled: false},
			},
			want: false,
		},
		{
			name: "reports false when the capability is absent",
			capabilities: []*cloud.Capability{
				{Name: string(CapabilitySourceOfTruth), Enabled: true},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Enabled(tt.capabilities, CapabilityMultiNamespaceSync))
		})
	}
}
