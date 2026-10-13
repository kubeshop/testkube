package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/localinstall"
)

func TestCanUseLocalContext_NeverTakesOverAnotherLogin(t *testing.T) {
	ours := config.CloudContext{OrganizationId: localinstall.DemoOrgID, ApiUri: "http://localhost:8090"}
	tests := map[string]struct {
		cfg  config.Data
		want bool
	}{
		"fresh config":         {config.Data{}, true},
		"our earlier install":  {config.Data{ContextType: config.ContextTypeCloud, CloudContext: ours}, true},
		"Testkube Cloud login": {config.Data{ContextType: config.ContextTypeCloud, CloudContext: config.CloudContext{OrganizationId: "tkcorg_acme", ApiUri: "https://api.testkube.io"}}, false},
		"chose kubeconfig":     {config.Data{ContextType: config.ContextTypeKubeconfig}, false},
		"demo org elsewhere":   {config.Data{ContextType: config.ContextTypeCloud, CloudContext: config.CloudContext{OrganizationId: localinstall.DemoOrgID, ApiUri: "https://testkube.acme.com"}}, false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, canUseLocalContext(tt.cfg))
		})
	}
}

func TestWithoutLocalContext_LeavesAnotherLoginAlone(t *testing.T) {
	cloud := config.Data{ContextType: config.ContextTypeCloud, CloudContext: config.CloudContext{OrganizationId: "tkcorg_acme", ApiUri: "https://api.testkube.io", ApiKey: "k"}}
	ours := localContext(config.Data{TelemetryEnabled: true}, localinstall.Ports{"api": 8090, "dashboard": 8080}, "tkcapi_x")

	kept, changed := withoutLocalContext(cloud)
	assert.False(t, changed)
	assert.Equal(t, cloud, kept)

	cleared, changed := withoutLocalContext(ours)
	assert.True(t, changed)
	assert.Empty(t, cleared.CloudContext.ApiKey)
	assert.True(t, canUseLocalContext(cleared), "a reinstall connects again")
	assert.True(t, cleared.TelemetryEnabled)
}

func TestLocalContext_FollowsMovedPortsAndKeepsOtherSettings(t *testing.T) {
	cfg := config.Data{TelemetryEnabled: true, Namespace: "testkube",
		ContextType: config.ContextTypeCloud, CloudContext: config.CloudContext{ApiUri: "http://localhost:8090", ApiKey: "old"}}

	got := localContext(cfg, localinstall.Ports{"api": 8091, "dashboard": 8081}, "tkcapi_new")

	assert.Equal(t, "http://localhost:8091", got.CloudContext.ApiUri)
	assert.Equal(t, "tkcapi_new", got.CloudContext.ApiKey)
	assert.True(t, isLocalContext(got), "the next run recognises it")
	assert.True(t, got.TelemetryEnabled, "settings outside the context stay")
}
