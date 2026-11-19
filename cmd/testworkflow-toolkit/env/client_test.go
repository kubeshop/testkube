package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxyTarget(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		wantService   string
		wantNamespace string
	}{
		{name: "keeps the execution namespace for a short name", host: "testkube-api-server", wantService: "testkube-api-server", wantNamespace: "exec"},
		{name: "reads the namespace of a name with a namespace", host: "testkube-api-server.agent", wantService: "testkube-api-server", wantNamespace: "agent"},
		{name: "reads the namespace of an absolute name", host: "testkube-api-server.agent.svc.cluster.local.", wantService: "testkube-api-server", wantNamespace: "agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, namespace := proxyTarget(tt.host, "exec")
			assert.Equal(t, tt.wantService, service)
			assert.Equal(t, tt.wantNamespace, namespace)
		})
	}
}
