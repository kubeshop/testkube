package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
)

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

func TestResolveNamedOrg(t *testing.T) {
	orgs := []cloudclient.Organization{{Id: testOrgID, Name: "platform"}}

	t.Run("no name leaves the organization untouched and never prompts", func(t *testing.T) {
		withSelector(t, false)
		master := config.Master{}

		err := ResolveNamedOrg("", "token", &master, false)

		assert.NoError(t, err)
		assert.Empty(t, master.OrgId)
	})

	t.Run("a name becomes an id", func(t *testing.T) {
		url := newControlPlane(t, orgs, nil)
		master := config.Master{OrgName: "platform"}

		err := ResolveNamedOrg(url, "token", &master, false)

		assert.NoError(t, err)
		assert.Equal(t, testOrgID, master.OrgId)
	})

	t.Run("an explicit id is left alone when no name is given", func(t *testing.T) {
		master := config.Master{OrgId: "tkcorg_explicit"}

		err := ResolveNamedOrg("", "token", &master, false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcorg_explicit", master.OrgId)
	})

	t.Run("a name that matches nothing is an error and leaves the id unset", func(t *testing.T) {
		url := newControlPlane(t, orgs, nil)
		master := config.Master{OrgName: "missing"}

		err := ResolveNamedOrg(url, "token", &master, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no organization named")
		assert.Empty(t, master.OrgId)
	})
}

func TestResolveOrgOrPrompt(t *testing.T) {
	orgs := []cloudclient.Organization{{Id: testOrgID, Name: "platform"}}

	t.Run("an id is used as given, without contacting the control plane", func(t *testing.T) {
		// No server: reaching the network at all would fail the test.
		orgID, err := ResolveOrgOrPrompt("", "token", config.Master{OrgId: "tkcorg_explicit"}, false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcorg_explicit", orgID)
	})

	t.Run("a name is resolved into an id", func(t *testing.T) {
		url := newControlPlane(t, orgs, nil)

		orgID, err := ResolveOrgOrPrompt(url, "token", config.Master{OrgName: "platform"}, false)

		assert.NoError(t, err)
		assert.Equal(t, testOrgID, orgID)
	})

	t.Run("an id wins over a name", func(t *testing.T) {
		// The flags are mutually exclusive at the cobra level, but the
		// precedence is what stops a stale name from overriding an explicit id.
		orgID, err := ResolveOrgOrPrompt("", "token", config.Master{
			OrgId:   "tkcorg_explicit",
			OrgName: "platform",
		}, false)

		assert.NoError(t, err)
		assert.Equal(t, "tkcorg_explicit", orgID)
	})

	t.Run("a name that matches nothing is an error", func(t *testing.T) {
		url := newControlPlane(t, orgs, nil)

		_, err := ResolveOrgOrPrompt(url, "token", config.Master{OrgName: "missing"}, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no organization named")
	})

	t.Run("a non-interactive terminal errors instead of prompting", func(t *testing.T) {
		withSelector(t, false)

		_, err := ResolveOrgOrPrompt("", "token", config.Master{}, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "--org-id or --org-name")
	})
}
