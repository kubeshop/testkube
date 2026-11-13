package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitOpsNamespaces(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    GitOpsNamespaces
		wantErr string
	}{
		{name: "empty value falls back to the default namespace", raw: "", want: GitOpsNamespaces{{Name: "testkube"}}},
		{name: "blank value falls back to the default namespace", raw: " , ", want: GitOpsNamespaces{{Name: "testkube"}}},
		{name: "blank and empty entries are ignored", raw: " a ,, b ,", want: GitOpsNamespaces{{Name: "a"}, {Name: "b"}}},
		{name: "ns=envId maps the namespace", raw: "a,b = env-1", want: GitOpsNamespaces{{Name: "a"}, {Name: "b", EnvID: "env-1"}}},
		{name: "* watches all", raw: "*", want: GitOpsNamespaces{{Name: "*"}}},
		{name: "* with ns=envId keeps the mapping", raw: "*,b=env-1", want: GitOpsNamespaces{{Name: "*"}, {Name: "b", EnvID: "env-1"}}},
		{name: "*=envId is an error", raw: "*=env-1", wantErr: `GITOPS_NAMESPACES: entry "*=env-1" cannot map "*" to an Environment`},
		{name: "duplicate namespace is an error", raw: "a,a=env-1", wantErr: `GITOPS_NAMESPACES: namespace "a" appears more than once`},
		{name: "empty envId after = is an error", raw: "a=", wantErr: `GITOPS_NAMESPACES: entry "a=" has no Environment id after '='`},
		{name: "empty namespace before = is an error", raw: "=env-1", wantErr: `GITOPS_NAMESPACES: entry "=env-1" has no namespace`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitOpsNamespaces(tt.raw, "testkube")
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitOpsNamespaces_WatchesAllAndNames(t *testing.T) {
	tests := []struct {
		name           string
		namespaces     GitOpsNamespaces
		wantWatchesAll bool
		wantNames      []string
	}{
		{name: "a * entry watches all and is left out of the names", namespaces: GitOpsNamespaces{{Name: "*"}, {Name: "a", EnvID: "env-1"}}, wantWatchesAll: true, wantNames: []string{"a"}},
		{name: "named entries only do not watch all", namespaces: GitOpsNamespaces{{Name: "a"}}, wantWatchesAll: false, wantNames: []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantWatchesAll, tt.namespaces.WatchesAll())
			assert.Equal(t, tt.wantNames, tt.namespaces.Names())
		})
	}
}

func TestValidateGitOpsNamespaces(t *testing.T) {
	proContext := func(capability bool) ProContext {
		return ProContext{
			EnvID: "env-token",
			Agent: ProContextAgent{Environments: []ProContextAgentEnvironment{{ID: "env-token"}, {ID: "env-other"}}},
			MultiNamespaceSyncSupportedInControlPlane: capability,
		}
	}
	tests := []struct {
		name       string
		namespaces GitOpsNamespaces
		capability bool
		wantErr    string
	}{
		{name: "default-only entries pass without the capability", namespaces: GitOpsNamespaces{{Name: "a"}, {Name: "*"}}},
		{name: "the token Environment passes without the capability", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-token"}}},
		{
			name:       "an unknown envId fails and lists the known Environments",
			namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-missing"}},
			capability: true,
			wantErr:    `GITOPS_NAMESPACES: namespace "a" maps to Environment "env-missing", which is not an Environment of the agent (known: env-token, env-other)`,
		},
		{
			name:       "a known envId without the capability fails and names the capability",
			namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-other"}},
			wantErr:    `GITOPS_NAMESPACES: namespace "a" maps to Environment "env-other", but the Control Plane lacks the sync-multi-namespace capability`,
		},
		{name: "a known envId with the capability passes", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-other"}}, capability: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitOpsNamespaces(tt.namespaces, proContext(tt.capability))
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
