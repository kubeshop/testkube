package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

func TestPopulateOrgAndEnvNames(t *testing.T) {
	t.Parallel()

	t.Run("fetches both names", func(t *testing.T) {
		t.Parallel()

		var requested []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requested = append(requested, r.URL.Path)
			switch r.URL.Path {
			case "/organizations/tkcorg_1":
				w.Write([]byte(`{"id":"tkcorg_1","name":"my-org"}`))
			case "/organizations/tkcorg_1/environments/tkcenv_1":
				w.Write([]byte(`{"id":"tkcenv_1","name":"my-env"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		cfg := config.Data{CloudContext: config.CloudContext{ApiKey: "tkcapi_1"}}

		cfg, err := PopulateOrgAndEnvNames(cfg, "tkcorg_1", "tkcenv_1", srv.URL)

		require.NoError(t, err)
		assert.Equal(t, "my-org", cfg.CloudContext.OrganizationName)
		assert.Equal(t, "my-env", cfg.CloudContext.EnvironmentName)
		assert.Equal(t, []string{"/organizations/tkcorg_1", "/organizations/tkcorg_1/environments/tkcenv_1"}, requested)
	})

	// Passing an organization without an environment resets the environment id,
	// so the environment lookup would be made with an empty id.
	t.Run("skips the environment lookup when no environment is set", func(t *testing.T) {
		t.Parallel()

		var requested []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requested = append(requested, r.URL.Path)
			w.Write([]byte(`{"id":"tkcorg_2","name":"other-org"}`))
		}))
		defer srv.Close()

		cfg := config.Data{CloudContext: config.CloudContext{
			ApiKey:          "tkcapi_1",
			EnvironmentId:   "tkcenv_old",
			EnvironmentName: "old-env",
		}}

		cfg, err := PopulateOrgAndEnvNames(cfg, "tkcorg_2", "", srv.URL)

		require.NoError(t, err)
		assert.Equal(t, "other-org", cfg.CloudContext.OrganizationName)
		assert.Empty(t, cfg.CloudContext.EnvironmentId)
		assert.Empty(t, cfg.CloudContext.EnvironmentName, "the name of a reset environment must not be kept")
		assert.Equal(t, []string{"/organizations/tkcorg_2"}, requested)
	})

	t.Run("makes no request when no ids are set", func(t *testing.T) {
		t.Parallel()

		var requested []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requested = append(requested, r.URL.Path)
		}))
		defer srv.Close()

		cfg := config.Data{CloudContext: config.CloudContext{
			ApiKey:           "tkcapi_1",
			OrganizationName: "stale-org",
			EnvironmentName:  "stale-env",
		}}

		cfg, err := PopulateOrgAndEnvNames(cfg, "", "", srv.URL)

		require.NoError(t, err)
		assert.Empty(t, cfg.CloudContext.OrganizationName)
		assert.Empty(t, cfg.CloudContext.EnvironmentName)
		assert.Empty(t, requested)
	})

	t.Run("reports the status the Control Plane answered with", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()

		cfg := config.Data{CloudContext: config.CloudContext{ApiKey: "tkcapi_expired"}}

		_, err := PopulateOrgAndEnvNames(cfg, "tkcorg_1", "tkcenv_1", srv.URL)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting organization")
		assert.Contains(t, err.Error(), "HTTP 401")
	})
}
