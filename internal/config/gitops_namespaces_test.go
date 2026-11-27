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
		{name: "one default namespace passes without the capability", namespaces: GitOpsNamespaces{{Name: "a"}}},
		{name: "the token Environment passes without the capability", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-token"}}},
		{
			name:       "two namespaces in one Environment without the capability fail",
			namespaces: GitOpsNamespaces{{Name: "a"}, {Name: "b", EnvID: "env-token"}},
			wantErr:    `GITOPS_NAMESPACES: 2 namespaces sync into Environment "env-token", but the Control Plane lacks the sync-multi-namespace capability`,
		},
		{
			name:       "* without the capability fails",
			namespaces: GitOpsNamespaces{{Name: "*"}},
			wantErr:    `GITOPS_NAMESPACES: "*" watches every namespace, but the Control Plane lacks the sync-multi-namespace capability`,
		},
		{name: "several namespaces and * with the capability pass", namespaces: GitOpsNamespaces{{Name: "a"}, {Name: "b"}, {Name: "*"}}, capability: true},
		{name: "the token Environment passes with the capability", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-token"}, {Name: "b"}}, capability: true},
		// The sync client sends every request into the token Environment, so a namespace mapped
		// to another one would sync there instead of where it says, even when the Control Plane
		// has the capability and the Environment is the agent's.
		{
			name:       "another Environment of the agent fails, with the capability",
			namespaces: GitOpsNamespaces{{Name: "a"}, {Name: "b", EnvID: "env-other"}},
			capability: true,
			wantErr:    `GITOPS_NAMESPACES: namespace "b" maps to Environment "env-other", but every namespace syncs into the Environment of the agent token, "env-token"`,
		},
		{
			name:       "another Environment of the agent fails, without the capability",
			namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-other"}},
			wantErr:    `GITOPS_NAMESPACES: namespace "a" maps to Environment "env-other", but every namespace syncs into the Environment of the agent token, "env-token"`,
		},
		{
			name:       "an unknown Environment fails",
			namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-missing"}},
			capability: true,
			wantErr:    `GITOPS_NAMESPACES: namespace "a" maps to Environment "env-missing", but every namespace syncs into the Environment of the agent token, "env-token"`,
		},
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
