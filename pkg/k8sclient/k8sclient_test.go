package k8sclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

func writeKubeconfig(t *testing.T, name, server string) string {
	t.Helper()
	content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: ` + server + `
  name: ` + name + `
contexts:
- context:
    cluster: ` + name + `
    user: ` + name + `
  name: ` + name + `
current-context: ` + name + `
users:
- name: ` + name + `
  user:
    token: ` + name + `-token
`
	p := filepath.Join(t.TempDir(), name+".yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGetK8sClientConfigMultipleKubeconfigFiles(t *testing.T) {
	one := writeKubeconfig(t, "cluster-one", "https://one.example.com")
	two := writeKubeconfig(t, "cluster-two", "https://two.example.com")
	sep := string(os.PathListSeparator)

	// Multiple files must merge instead of being treated as one path
	// (https://github.com/kubeshop/testkube/issues/657); the first file's
	// current-context wins, matching kubectl precedence.
	t.Setenv("KUBECONFIG", one+sep+two)
	config, err := GetK8sClientConfig()
	assert.NoError(t, err)
	assert.Equal(t, "https://one.example.com", config.Host)

	t.Setenv("KUBECONFIG", two+sep+one)
	config, err = GetK8sClientConfig()
	assert.NoError(t, err)
	assert.Equal(t, "https://two.example.com", config.Host)
}

func TestGetK8sClientConfigSingleKubeconfigFile(t *testing.T) {
	one := writeKubeconfig(t, "cluster-one", "https://one.example.com")

	t.Setenv("KUBECONFIG", one)
	config, err := GetK8sClientConfig()
	assert.NoError(t, err)
	assert.Equal(t, "https://one.example.com", config.Host)
}

func TestGetClusterVersion(t *testing.T) {
	client := fake.NewClientset()

	v, err := GetClusterVersion(client)
	assert.NoError(t, err)
	assert.Equal(t, "v0.0.0-master+$Format:%H$", v)
}

func TestGetAPIServerLogs(t *testing.T) {
	client := fake.NewClientset()

	logs, err := GetAPIServerLogs(context.Background(), client, "testkube")
	assert.NoError(t, err)
	assert.Equal(t, []string([]string{}), logs)
}

func TestGetOperatorLogs(t *testing.T) {
	client := fake.NewClientset()

	logs, err := GetOperatorLogs(context.Background(), client, "testkube")
	assert.NoError(t, err)
	assert.Equal(t, []string([]string{}), logs)
}

func TestGetPodLogs(t *testing.T) {
	client := fake.NewClientset()

	logs, err := GetPodLogs(context.Background(), client, "testkube", "selector")
	assert.NoError(t, err)
	assert.Equal(t, []string([]string{}), logs)
}
