package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

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

// withSelector pins whether the org/environment selectors consider the terminal
// interactive, so tests do not depend on how they are run.
func withSelector(t *testing.T, interactive bool) {
	t.Helper()
	original := selectorInteractive
	selectorInteractive = func() bool { return interactive }
	t.Cleanup(func() { selectorInteractive = original })
}
