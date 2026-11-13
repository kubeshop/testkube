package config

import (
	"testing"

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
		{name: "*=envId is an error", raw: "*=env-1", wantErr: "cannot map"},
		{name: "duplicate namespace is an error", raw: "a,a=env-1", wantErr: "more than once"},
		{name: "empty envId after = is an error", raw: "a=", wantErr: "no Environment id"},
		{name: "empty namespace before = is an error", raw: "=env-1", wantErr: "no namespace"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitOpsNamespaces(tt.raw, "testkube")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.ErrorContains(t, err, "GITOPS_NAMESPACES:")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGitOpsNamespacesNamesAndWatchesAll(t *testing.T) {
	ns := GitOpsNamespaces{{Name: "*"}, {Name: "a", EnvID: "env-1"}}
	require.True(t, ns.WatchesAll())
	require.Equal(t, []string{"a"}, ns.Names())

	ns = GitOpsNamespaces{{Name: "a"}}
	require.False(t, ns.WatchesAll())
	require.Equal(t, []string{"a"}, ns.Names())
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
		wantErr    []string
	}{
		{name: "default-only entries pass without the capability", namespaces: GitOpsNamespaces{{Name: "a"}, {Name: "*"}}},
		{name: "the token Environment passes without the capability", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-token"}}},
		{name: "an unknown envId fails and names it", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-missing"}}, capability: true, wantErr: []string{"env-missing", `"a"`, "env-other"}},
		{name: "a known envId without the capability fails and names the capability", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-other"}}, wantErr: []string{"capability", "sync-multi-namespace"}},
		{name: "a known envId with the capability passes", namespaces: GitOpsNamespaces{{Name: "a", EnvID: "env-other"}}, capability: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitOpsNamespaces(tt.namespaces, proContext(tt.capability))
			if len(tt.wantErr) == 0 {
				require.NoError(t, err)
				return
			}
			for _, want := range tt.wantErr {
				require.ErrorContains(t, err, want)
			}
		})
	}
}
