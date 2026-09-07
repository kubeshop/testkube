package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

const testOrgID = "tkcorg_1111111111111111"

// newControlPlane serves the organization and environment list endpoints the
// resolvers read, for the single organization used across these tests.
func newControlPlane(t *testing.T, orgs []cloudclient.Organization, envs []cloudclient.Environment) string {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/organizations", func(w http.ResponseWriter, r *http.Request) {
		writeElements(t, w, orgs)
	})
	mux.HandleFunc("/organizations/"+testOrgID+"/environments", func(w http.ResponseWriter, r *http.Request) {
		writeElements(t, w, envs)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func writeElements[T any](t *testing.T, w http.ResponseWriter, elements []T) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(map[string][]T{"elements": elements}))
}

func TestResolveOrgID(t *testing.T) {
	orgs := []cloudclient.Organization{
		{Id: testOrgID, Name: "platform"},
		{Id: "tkcorg_2222222222222222", Name: "payments"},
	}

	t.Run("exact name resolves to its id", func(t *testing.T) {
		url := newControlPlane(t, orgs, nil)

		id, err := ResolveOrgID(url, "token", "platform")

		assert.NoError(t, err)
		assert.Equal(t, testOrgID, id)
	})

	t.Run("match is case sensitive", func(t *testing.T) {
		url := newControlPlane(t, orgs, nil)

		_, err := ResolveOrgID(url, "token", "Platform")

		require.Error(t, err)
		assert.Contains(t, err.Error(), `no organization named "Platform"`)
		// The error lists what the caller could have passed instead.
		assert.Contains(t, err.Error(), "platform, payments")
	})

	t.Run("duplicate names are an error rather than an arbitrary pick", func(t *testing.T) {
		url := newControlPlane(t, []cloudclient.Organization{
			{Id: testOrgID, Name: "platform"},
			{Id: "tkcorg_2222222222222222", Name: "platform"},
		}, nil)

		_, err := ResolveOrgID(url, "token", "platform")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ambiguous")
		assert.Contains(t, err.Error(), "--org-id")
		assert.Contains(t, err.Error(), "tkcorg_2222222222222222")
	})

	t.Run("no organizations at all", func(t *testing.T) {
		url := newControlPlane(t, nil, nil)

		_, err := ResolveOrgID(url, "token", "platform")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no organizations available")
	})
}

func TestResolveEnvID(t *testing.T) {
	envs := []cloudclient.Environment{
		{Id: "tkcenv_1111111111111111", Name: "staging", Slug: "staging-slug"},
		{Id: "tkcenv_2222222222222222", Name: "production", Slug: "production-slug"},
	}

	t.Run("exact name resolves to its id", func(t *testing.T) {
		url := newControlPlane(t, nil, envs)

		id, err := ResolveEnvID(url, "token", testOrgID, "staging")

		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_1111111111111111", id)
	})

	t.Run("falls back to the slug when no name matches", func(t *testing.T) {
		url := newControlPlane(t, nil, envs)

		id, err := ResolveEnvID(url, "token", testOrgID, "production-slug")

		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_2222222222222222", id)
	})

	t.Run("a name match wins over another environment's slug", func(t *testing.T) {
		url := newControlPlane(t, nil, []cloudclient.Environment{
			{Id: "tkcenv_1111111111111111", Name: "staging", Slug: "old-staging"},
			{Id: "tkcenv_2222222222222222", Name: "staging-next", Slug: "staging"},
		})

		id, err := ResolveEnvID(url, "token", testOrgID, "staging")

		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_1111111111111111", id)
	})

	t.Run("empty slugs never match an empty name", func(t *testing.T) {
		url := newControlPlane(t, nil, []cloudclient.Environment{
			{Id: "tkcenv_1111111111111111", Name: "staging"},
		})

		_, err := ResolveEnvID(url, "token", testOrgID, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no environment named")
	})

	t.Run("duplicate names are an error", func(t *testing.T) {
		url := newControlPlane(t, nil, []cloudclient.Environment{
			{Id: "tkcenv_1111111111111111", Name: "staging"},
			{Id: "tkcenv_2222222222222222", Name: "staging"},
		})

		_, err := ResolveEnvID(url, "token", testOrgID, "staging")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ambiguous")
		assert.Contains(t, err.Error(), "--env-id")
	})
}

func TestResolveOrgAndEnvIDs(t *testing.T) {
	orgs := []cloudclient.Organization{{Id: testOrgID, Name: "platform"}}
	envs := []cloudclient.Environment{{Id: "tkcenv_1111111111111111", Name: "staging"}}

	t.Run("ids are used as given, without contacting the control plane", func(t *testing.T) {
		// No server: reaching the network at all would fail the test.
		orgID, envID, err := ResolveOrgAndEnvIDs("", "token", config.Master{
			OrgId: "tkcorg_explicit",
			EnvId: "tkcenv_explicit",
		}, false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcorg_explicit", orgID)
		assert.Equal(t, "tkcenv_explicit", envID)
	})

	t.Run("names are resolved into ids", func(t *testing.T) {
		url := newControlPlane(t, orgs, envs)

		orgID, envID, err := ResolveOrgAndEnvIDs(url, "token", config.Master{
			OrgName: "platform",
			EnvName: "staging",
		}, false)

		assert.NoError(t, err)
		assert.Equal(t, testOrgID, orgID)
		assert.Equal(t, "tkcenv_1111111111111111", envID)
	})

	t.Run("an org id combines with an env name", func(t *testing.T) {
		url := newControlPlane(t, orgs, envs)

		orgID, envID, err := ResolveOrgAndEnvIDs(url, "token", config.Master{
			OrgId:   testOrgID,
			EnvName: "staging",
		}, false)

		assert.NoError(t, err)
		assert.Equal(t, testOrgID, orgID)
		assert.Equal(t, "tkcenv_1111111111111111", envID)
	})

	t.Run("a bad org name does not fall through to the environment lookup", func(t *testing.T) {
		url := newControlPlane(t, orgs, envs)

		_, _, err := ResolveOrgAndEnvIDs(url, "token", config.Master{
			OrgName: "missing",
			EnvName: "staging",
		}, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no organization named")
	})

	t.Run("a non-interactive terminal errors instead of prompting", func(t *testing.T) {
		withSelector(t, false)

		_, _, err := ResolveOrgAndEnvIDs("", "token", config.Master{}, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "--org-id or --org-name")
	})

	t.Run("a non-interactive terminal errors on a missing environment too", func(t *testing.T) {
		withSelector(t, false)

		_, _, err := ResolveOrgAndEnvIDs("", "token", config.Master{OrgId: testOrgID}, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "--env-id or --env-name")
	})
}

func TestResolveNamedOrgAndEnv(t *testing.T) {
	orgs := []cloudclient.Organization{{Id: testOrgID, Name: "platform"}}
	envs := []cloudclient.Environment{{Id: "tkcenv_1111111111111111", Name: "staging"}}

	t.Run("no names leaves everything untouched and never prompts", func(t *testing.T) {
		withSelector(t, false)
		master := config.Master{}

		err := ResolveNamedOrgAndEnv("", "token", &master, "", false)

		assert.NoError(t, err)
		assert.Empty(t, master.OrgId)
		assert.Empty(t, master.EnvId)
	})

	t.Run("names become ids", func(t *testing.T) {
		url := newControlPlane(t, orgs, envs)
		master := config.Master{OrgName: "platform", EnvName: "staging"}

		err := ResolveNamedOrgAndEnv(url, "token", &master, "", false)

		assert.NoError(t, err)
		assert.Equal(t, testOrgID, master.OrgId)
		assert.Equal(t, "tkcenv_1111111111111111", master.EnvId)
	})

	t.Run("an env name alone resolves against the organization already in context", func(t *testing.T) {
		url := newControlPlane(t, orgs, envs)
		master := config.Master{EnvName: "staging"}

		err := ResolveNamedOrgAndEnv(url, "token", &master, testOrgID, false)

		assert.NoError(t, err)
		assert.Empty(t, master.OrgId, "the fallback organization is only for the lookup")
		assert.Equal(t, "tkcenv_1111111111111111", master.EnvId)
	})

	t.Run("an env name with no organization anywhere is an error", func(t *testing.T) {
		master := config.Master{EnvName: "staging"}

		err := ResolveNamedOrgAndEnv("", "token", &master, "", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "without an organization")
	})
}

// withSelector pins whether the org/environment selectors consider the terminal
// interactive, so tests do not depend on how they are run.
func withSelector(t *testing.T, interactive bool) {
	t.Helper()
	original := selectorInteractive
	selectorInteractive = func() bool { return interactive }
	t.Cleanup(func() { selectorInteractive = original })
}
