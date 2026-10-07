package localinstall

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeKind(clusters string, calls *[]string) runFunc {
	return func(_ context.Context, _ string, args ...string) ([]byte, error) {
		*calls = append(*calls, strings.Join(args, " "))
		if args[0] == "get" {
			return []byte(clusters), nil
		}
		return nil, nil
	}
}

func TestClusterEnsure_ReusesOursOnly(t *testing.T) {
	tests := []struct {
		name        string
		clusters    string
		wantCreated bool
		wantCall    string
	}{
		{"our cluster is reused, not recreated", "kind\ntestkube\n", false,
			"export kubeconfig --name testkube --kubeconfig /home/u/.testkube/kubeconfig"},
		{"a similar name is someone else's", "testkube-dev\n", true,
			"create cluster --name testkube --image " + nodeImage + " --kubeconfig /home/u/.testkube/kubeconfig --wait 2m"},
		{"no clusters at all", "No kind clusters found.\n", true,
			"create cluster --name testkube --image " + nodeImage + " --kubeconfig /home/u/.testkube/kubeconfig --wait 2m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			c := &Cluster{kind: "kind", kubeconfig: "/home/u/.testkube/kubeconfig", run: fakeKind(tt.clusters, &calls)}

			created, _, err := c.Ensure(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tt.wantCreated, created)
			assert.Equal(t, []string{"get clusters", tt.wantCall}, calls)
		})
	}
}
