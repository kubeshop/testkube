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
	const agentNamespace = "testkube"
	tests := []struct {
		name       string
		namespaces GitOpsNamespaces
		capability bool
		// envOverride is TESTKUBE_PRO_ENV_ID, which ReadProContext keeps as EnvID over the
		// Environment of the token.
		envOverride string
		// syncEnvID is the Environment the Control Plane reports for the token; "-" means none.
		syncEnvID string
		wantErr   string
	}{
		{name: "the agent namespace passes without the capability", namespaces: GitOpsNamespaces{{Name: agentNamespace}}},
		{name: "the token Environment passes without the capability", namespaces: GitOpsNamespaces{{Name: agentNamespace, EnvID: "env-token"}}},
		// Without the capability the Control Plane deletes in the agent namespace but updates in
		// the payload namespace, so a resource synced from another namespace may never be deleted.
		{
			name:       "one other namespace without the capability fails",
			namespaces: GitOpsNamespaces{{Name: "team-a"}},
			wantErr:    `GITOPS_NAMESPACES: namespace "team-a" is not the agent namespace "testkube", but the Control Plane lacks the sync-multi-namespace capability`,
		},
		{
			name:       "two namespaces without the capability fail",
			namespaces: GitOpsNamespaces{{Name: agentNamespace}, {Name: "team-b", EnvID: "env-token"}},
			wantErr:    `GITOPS_NAMESPACES: namespace "team-b" is not the agent namespace "testkube", but the Control Plane lacks the sync-multi-namespace capability`,
		},
		{
			name:       "* without the capability fails",
			namespaces: GitOpsNamespaces{{Name: "*"}},
			wantErr:    `GITOPS_NAMESPACES: "*" watches every namespace, but the Control Plane lacks the sync-multi-namespace capability`,
		},
		{name: "one other namespace with the capability passes", namespaces: GitOpsNamespaces{{Name: "team-a"}}, capability: true},
		{name: "several namespaces and * with the capability pass", namespaces: GitOpsNamespaces{{Name: "a"}, {Name: "b"}, {Name: "*"}}, capability: true},
		{name: "the token Environment passes with the capability", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-token"}, {Name: "b"}}, capability: true},
		// The sync client sends every request into the Environment of the token, so a namespace
		// mapped to another one would sync there instead of where it says.
		{
			name:       "another Environment fails, with the capability",
			namespaces: GitOpsNamespaces{{Name: "a"}, {Name: "b", EnvID: "env-other"}},
			capability: true,
			wantErr:    `GITOPS_NAMESPACES: namespace "b" maps to Environment "env-other", but every namespace syncs into the Environment of the agent token, "env-token"`,
		},
		{
			name:       "another Environment fails, without the capability",
			namespaces: GitOpsNamespaces{{Name: agentNamespace, EnvID: "env-other"}},
			wantErr:    `GITOPS_NAMESPACES: namespace "testkube" maps to Environment "env-other", but every namespace syncs into the Environment of the agent token, "env-token"`,
		},
		{
			name:        "a TESTKUBE_PRO_ENV_ID override does not move the sync Environment",
			namespaces:  GitOpsNamespaces{{Name: "team-b", EnvID: "env-other"}},
			capability:  true,
			envOverride: "env-other",
			wantErr:     `GITOPS_NAMESPACES: namespace "team-b" maps to Environment "env-other", but every namespace syncs into the Environment of the agent token, "env-token"`,
		},
		{
			name:       "a mapping fails when the Control Plane reports no Environment",
			namespaces: GitOpsNamespaces{{Name: agentNamespace, EnvID: "env-token"}},
			syncEnvID:  "-",
			wantErr:    `GITOPS_NAMESPACES: namespace "testkube" maps to Environment "env-token", but the Control Plane reported no Environment for the agent token to check it against`,
		},
		{name: "no mapping passes when the Control Plane reports no Environment", namespaces: GitOpsNamespaces{{Name: agentNamespace}}, syncEnvID: "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncEnvID := "env-token"
			if tt.syncEnvID == "-" {
				syncEnvID = ""
			}
			envID := syncEnvID
			if tt.envOverride != "" {
				envID = tt.envOverride
			}
			proContext := ProContext{
				EnvID:             envID,
				ControlPlaneEnvID: syncEnvID,
				MultiNamespaceSyncSupportedInControlPlane: tt.capability,
			}
			err := ValidateGitOpsNamespaces(tt.namespaces, agentNamespace, proContext)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
