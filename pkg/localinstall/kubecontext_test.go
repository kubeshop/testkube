package localinstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Offering "your cluster" needs a context that really exists.
func TestCurrentKubeContext_OnlyAnExistingContextCounts(t *testing.T) {
	contexts := "contexts:\n- name: gke_acme_prod\n  context: {cluster: c, user: u}\n- name: kind-dev\n  context: {cluster: c, user: u}\n"
	tests := map[string]struct {
		kubeconfig string
		wantName   string
		wantLocal  bool
	}{
		"no kubeconfig":           {"", "", false},
		"no current context":      {contexts, "", false},
		"current context missing": {contexts + "current-context: gone\n", "", false},
		"remote cluster":          {contexts + "current-context: gke_acme_prod\n", "gke_acme_prod", false},
		"local cluster":           {contexts + "current-context: kind-dev\n", "kind-dev", true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config")
			if tt.kubeconfig != "" {
				if err := os.WriteFile(path, []byte("apiVersion: v1\nkind: Config\n"+tt.kubeconfig), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("KUBECONFIG", path)

			gotName, gotLocal := CurrentKubeContext()

			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantLocal, gotLocal)
		})
	}
}
