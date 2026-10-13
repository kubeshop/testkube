package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/localinstall"
)

func markedKeys(t *testing.T, keys ...string) {
	saved := isOurCLIKey
	t.Cleanup(func() { isOurCLIKey = saved })
	isOurCLIKey = func(key string) bool {
		for _, k := range keys {
			if k == key {
				return true
			}
		}
		return false
	}
}

func TestCanUseLocalContext_NeverTakesOverAnotherLogin(t *testing.T) {
	markedKeys(t, "tkcapi_ours")
	ours := config.CloudContext{OrganizationId: localinstall.DemoOrgID, ApiUri: "http://localhost:8090", ApiKey: "tkcapi_ours"}
	otherDemo := config.CloudContext{OrganizationId: localinstall.DemoOrgID, ApiUri: "http://localhost:8090", ApiKey: "tkcapi_theirs"}
	tests := map[string]struct {
		cfg      config.Data
		firstRun bool
		want     bool
	}{
		"first CLI run ever":          {config.Data{}, true, true},
		"OSS user on kubeconfig mode": {config.Data{}, false, false},
		"our earlier install":         {config.Data{ContextType: config.ContextTypeCloud, CloudContext: ours}, false, true},
		"Testkube Cloud login":        {config.Data{ContextType: config.ContextTypeCloud, CloudContext: config.CloudContext{OrganizationId: "tkcorg_acme", ApiUri: "https://api.testkube.io"}}, false, false},
		"chose kubeconfig":            {config.Data{ContextType: config.ContextTypeKubeconfig}, false, false},
		"other demo on the same port": {config.Data{ContextType: config.ContextTypeCloud, CloudContext: otherDemo}, false, false},
		"demo org elsewhere":          {config.Data{ContextType: config.ContextTypeCloud, CloudContext: config.CloudContext{OrganizationId: localinstall.DemoOrgID, ApiUri: "https://testkube.acme.com"}}, false, false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, canUseLocalContext(tt.cfg, tt.firstRun))
		})
	}
}

func TestWithoutLocalContext_LeavesAnotherLoginAlone(t *testing.T) {
	markedKeys(t, "tkcapi_x")
	cloud := config.Data{ContextType: config.ContextTypeCloud, CloudContext: config.CloudContext{OrganizationId: "tkcorg_acme", ApiUri: "https://api.testkube.io", ApiKey: "k"}}
	ours := localContext(config.Data{TelemetryEnabled: true}, localinstall.Ports{"api": 8090, "dashboard": 8080}, "tkcapi_x")

	kept, changed := withoutLocalContext(cloud)
	assert.False(t, changed)
	assert.Equal(t, cloud, kept)

	cleared, changed := withoutLocalContext(ours)
	assert.True(t, changed)
	assert.Empty(t, cleared.CloudContext.ApiKey)
	assert.True(t, canUseLocalContext(cleared, true), "a reinstall on a fresh CLI connects again")
	assert.True(t, cleared.TelemetryEnabled)
}

func TestLocalContext_FollowsMovedPortsAndKeepsOtherSettings(t *testing.T) {
	markedKeys(t, "tkcapi_new")
	cfg := config.Data{TelemetryEnabled: true, Namespace: "testkube",
		ContextType: config.ContextTypeCloud, CloudContext: config.CloudContext{ApiUri: "http://localhost:8090", ApiKey: "old"}}

	got := localContext(cfg, localinstall.Ports{"api": 8091, "dashboard": 8081}, "tkcapi_new")

	assert.Equal(t, "http://localhost:8091", got.CloudContext.ApiUri)
	assert.Equal(t, "tkcapi_new", got.CloudContext.ApiKey)
	assert.True(t, isLocalContext(got), "the next run recognises it")
	assert.True(t, got.TelemetryEnabled, "settings outside the context stay")
}
