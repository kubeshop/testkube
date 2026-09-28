package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

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

func TestResolveNamedEnv(t *testing.T) {
	envs := []cloudclient.Environment{{Id: "tkcenv_1111111111111111", Name: "staging"}}

	t.Run("no name leaves the environment untouched and never prompts", func(t *testing.T) {
		withSelector(t, false)
		master := config.Master{}

		err := ResolveNamedEnv("", "token", &master, "", false)

		assert.NoError(t, err)
		assert.Empty(t, master.EnvId)
	})

	t.Run("a name becomes an id, scoped to the resolved organization", func(t *testing.T) {
		url := newControlPlane(t, nil, envs)
		master := config.Master{OrgId: testOrgID, EnvName: "staging"}

		err := ResolveNamedEnv(url, "token", &master, "", false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_1111111111111111", master.EnvId)
	})

	t.Run("a name alone resolves against the organization already in context", func(t *testing.T) {
		url := newControlPlane(t, nil, envs)
		master := config.Master{EnvName: "staging"}

		err := ResolveNamedEnv(url, "token", &master, testOrgID, false)

		assert.NoError(t, err)
		assert.Empty(t, master.OrgId, "the fallback organization is only for the lookup")
		assert.Equal(t, "tkcenv_1111111111111111", master.EnvId)
	})

	t.Run("an organization set on master wins over the fallback", func(t *testing.T) {
		url := newControlPlane(t, nil, envs)
		master := config.Master{OrgId: testOrgID, EnvName: "staging"}

		// The fallback names an organization the stub serves no environments
		// for, so reaching for it instead would fail the lookup.
		err := ResolveNamedEnv(url, "token", &master, "tkcorg_9999999999999999", false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_1111111111111111", master.EnvId)
	})

	t.Run("a name with no organization anywhere is an error", func(t *testing.T) {
		master := config.Master{EnvName: "staging"}

		err := ResolveNamedEnv("", "token", &master, "", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "without an organization")
	})
}

func TestResolveEnvOrPrompt(t *testing.T) {
	envs := []cloudclient.Environment{{Id: "tkcenv_1111111111111111", Name: "staging"}}

	t.Run("an id is used as given, without contacting the control plane", func(t *testing.T) {
		// No server: reaching the network at all would fail the test.
		envID, err := ResolveEnvOrPrompt("", "token", testOrgID, config.Master{EnvId: "tkcenv_explicit"}, false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_explicit", envID)
	})

	t.Run("a name is resolved into an id within the given organization", func(t *testing.T) {
		url := newControlPlane(t, nil, envs)

		envID, err := ResolveEnvOrPrompt(url, "token", testOrgID, config.Master{EnvName: "staging"}, false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_1111111111111111", envID)
	})

	t.Run("an id wins over a name", func(t *testing.T) {
		envID, err := ResolveEnvOrPrompt("", "token", testOrgID, config.Master{
			EnvId:   "tkcenv_explicit",
			EnvName: "staging",
		}, false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcenv_explicit", envID)
	})

	t.Run("a name that matches nothing is an error", func(t *testing.T) {
		url := newControlPlane(t, nil, envs)

		_, err := ResolveEnvOrPrompt(url, "token", testOrgID, config.Master{EnvName: "missing"}, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no environment named")
	})

	t.Run("a non-interactive terminal errors instead of prompting", func(t *testing.T) {
		withSelector(t, false)

		_, err := ResolveEnvOrPrompt("", "token", testOrgID, config.Master{}, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "--env-id or --env-name")
	})
}
