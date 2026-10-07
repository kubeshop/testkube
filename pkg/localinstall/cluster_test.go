package localinstall

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dockerPortOutput = `30080/tcp -> 127.0.0.1:8080
30090/tcp -> 127.0.0.1:8090
30556/tcp -> 127.0.0.1:5556
30900/tcp -> 127.0.0.1:9001
30990/tcp -> 127.0.0.1:9090
`

const mounts = "/var/lib/docker\n/var/local-path-provisioner\n"

func fakeKind(clusters, dockerPort string, calls *[]string) runFunc {
	return fakeKindWithMounts(clusters, dockerPort, mounts, calls)
}

func fakeKindWithMounts(clusters, dockerPort, mounts string, calls *[]string) runFunc {
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		*calls = append(*calls, name+" "+strings.Join(args, " "))
		switch {
		case args[0] == "get":
			return []byte(clusters), nil
		case name == "docker" && args[0] == "port":
			return []byte(dockerPort), nil
		case name == "docker" && args[0] == "inspect":
			return []byte(mounts), nil
		}
		return nil, nil
	}
}

func allFree(int) bool { return true }

func TestClusterEnsure_ReusesOursOnly(t *testing.T) {
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "kubeconfig")
	create := "kind create cluster --name testkube --config " + filepath.Join(dir, "kind.yaml") +
		" --image " + nodeImage + " --kubeconfig " + kubeconfig + " --wait 2m"
	tests := []struct {
		name        string
		clusters    string
		wantCreated bool
		wantCalls   []string
	}{
		{"our cluster is reused, not recreated", "kind\ntestkube\n", false, []string{
			"kind get clusters",
			"kind export kubeconfig --name testkube --kubeconfig " + kubeconfig,
			"docker port testkube-control-plane",
			"docker inspect -f {{range .Mounts}}{{.Destination}}\n{{end}} testkube-control-plane",
		}},
		{"a similar name is someone else's", "testkube-dev\n", true, []string{"kind get clusters", create}},
		{"no clusters at all", "No kind clusters found.\n", true, []string{"kind get clusters", create}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			c := &Cluster{kind: "kind", dir: dir, kubeconfig: kubeconfig, portFree: allFree,
				run: fakeKind(tt.clusters, dockerPortOutput, &calls)}

			state, _, err := c.Ensure(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tt.wantCreated, state.Created)
			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestClusterEnsure_ReusedClusterKeepsItsPorts(t *testing.T) {
	var calls []string
	c := &Cluster{kind: "kind", dir: t.TempDir(), run: fakeKind("testkube\n", dockerPortOutput, &calls)}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 9001, state.Ports["storage"])
	assert.Equal(t, []string{"storage 9000→9001"}, state.Ports.Moved())
}

func TestClusterEnsure_OlderClusterIsRejected(t *testing.T) {
	for name, run := range map[string]func(*[]string) runFunc{
		"no ports": func(c *[]string) runFunc { return fakeKind("testkube\n", "", c) },
		"no data folder": func(c *[]string) runFunc {
			return fakeKindWithMounts("testkube\n", dockerPortOutput, "/var/lib/docker\n", c)
		},
	} {
		t.Run(name, func(t *testing.T) {
			var calls []string
			c := &Cluster{kind: "kind", dir: t.TempDir(), run: run(&calls)}

			_, _, err := c.Ensure(context.Background())

			assert.ErrorIs(t, err, ErrClusterOutdated)
		})
	}
}

func TestPickPorts_MovesBusyPortsWithoutCollisions(t *testing.T) {
	busy := map[int]bool{8080: true, 8081: true, 8082: true, 8083: true, 8084: true,
		8085: true, 8086: true, 8087: true, 8088: true, 8089: true, 9000: true}

	ports := pickPorts(func(p int) bool { return !busy[p] })

	assert.Equal(t, Ports{"dashboard": 8090, "api": 8091, "login": 5556, "storage": 9001, "ai": 9090}, ports)
}

func TestClusterEnsure_PortsListenOnThisMachineOnly(t *testing.T) {
	dir := t.TempDir()
	var calls []string
	c := &Cluster{kind: "kind", dir: dir, portFree: allFree, run: fakeKind("", "", &calls)}

	_, _, err := c.Ensure(context.Background())
	require.NoError(t, err)

	config, err := os.ReadFile(filepath.Join(dir, "kind.yaml"))
	require.NoError(t, err)
	assert.Equal(t, len(clusterPorts), strings.Count(string(config), "hostPort:"))
	assert.Equal(t, len(clusterPorts), strings.Count(string(config), "listenAddress: 127.0.0.1"))
}

func TestClusterEnsure_DataFolderIsMountedAndWritableByPods(t *testing.T) {
	dir := t.TempDir()
	var calls []string
	c := &Cluster{kind: "kind", dir: dir, portFree: allFree, run: fakeKind("", "", &calls)}

	_, _, err := c.Ensure(context.Background())
	require.NoError(t, err)

	config, _ := os.ReadFile(filepath.Join(dir, "kind.yaml"))
	assert.Contains(t, string(config), fmt.Sprintf("hostPath: %q\n    containerPath: /var/local-path-provisioner", filepath.Join(dir, "data")))
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "data"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o777), info.Mode().Perm())
	}
}
