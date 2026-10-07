package localinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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

type kindFake struct {
	clusters, ports, mounts, running string
	startErr                         error
	refusedWaits                     int
	calls                            []string
}

func (f *kindFake) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	switch {
	case name != "docker" && args[0] == "get":
		return []byte(f.clusters), nil
	case args[0] == "port":
		return []byte(f.ports), nil
	case args[0] == "inspect" && args[2] == "{{.State.Running}}":
		return []byte(f.running), nil
	case args[0] == "inspect":
		return []byte(f.mounts), nil
	case args[0] == "start":
		return []byte("Bind for 127.0.0.1:9000 failed: port is already allocated"), f.startErr
	case args[0] == "exec" && f.refusedWaits > 0:
		f.refusedWaits--
		return []byte("The connection to the server testkube-control-plane:6443 was refused"), errors.New("exit status 1")
	}
	return nil, nil
}

// A healthy, running cluster of ours unless a test overrides it.
func ours() *kindFake {
	return &kindFake{clusters: "testkube\n", ports: dockerPortOutput, mounts: mounts, running: "true\n"}
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
			"docker inspect -f {{.State.Running}} testkube-control-plane",
			"kind export kubeconfig --name testkube --kubeconfig " + kubeconfig,
			"docker port testkube-control-plane",
			"docker inspect -f {{range .Mounts}}{{.Destination}}\n{{end}} testkube-control-plane",
		}},
		{"a similar name is someone else's", "testkube-dev\n", true, []string{"kind get clusters", create}},
		{"no clusters at all", "No kind clusters found.\n", true, []string{"kind get clusters", create}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := ours()
			fake.clusters = tt.clusters
			c := &Cluster{kind: "kind", dir: dir, kubeconfig: kubeconfig, portFree: allFree, run: fake.run}

			state, _, err := c.Ensure(context.Background())

			require.NoError(t, err)
			assert.Equal(t, tt.wantCreated, state.Created)
			assert.Equal(t, tt.wantCalls, fake.calls)
		})
	}
}

func TestClusterEnsure_ReusedClusterKeepsItsPorts(t *testing.T) {
	c := &Cluster{kind: "kind", dir: t.TempDir(), run: ours().run}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 9001, state.Ports["storage"])
	assert.Equal(t, []string{"storage 9000→9001"}, state.Ports.Moved())
}

func TestClusterEnsure_OlderClusterIsRejected(t *testing.T) {
	noPorts, noData := ours(), ours()
	noPorts.ports, noData.mounts = "", "/var/lib/docker\n"
	for name, fake := range map[string]*kindFake{"no ports": noPorts, "no data folder": noData} {
		t.Run(name, func(t *testing.T) {
			c := &Cluster{kind: "kind", dir: t.TempDir(), run: fake.run}

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
	c := &Cluster{kind: "kind", dir: dir, portFree: allFree, run: (&kindFake{}).run}

	_, _, err := c.Ensure(context.Background())
	require.NoError(t, err)

	config, err := os.ReadFile(filepath.Join(dir, "kind.yaml"))
	require.NoError(t, err)
	assert.Equal(t, len(clusterPorts), strings.Count(string(config), "hostPort:"))
	assert.Equal(t, len(clusterPorts), strings.Count(string(config), "listenAddress: 127.0.0.1"))
}

func TestClusterEnsure_DataFolderIsMountedAndWritableByPods(t *testing.T) {
	dir := t.TempDir()
	c := &Cluster{kind: "kind", dir: dir, portFree: allFree, run: (&kindFake{}).run}

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

func TestClusterEnsure_StoppedClusterIsStartedBeforeReading(t *testing.T) {
	fake := ours()
	fake.running = "false\n"
	c := &Cluster{kind: "kind", dir: t.TempDir(), run: fake.run}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.True(t, state.Started)
	assert.Equal(t, "docker start testkube-control-plane", fake.calls[2])
	assert.Contains(t, fake.calls[3], "wait --for=condition=Ready")
}

func TestClusterEnsure_WaitsWhileTheAPIIsStillStarting(t *testing.T) {
	startRetry = time.Millisecond
	t.Cleanup(func() { startRetry = 2 * time.Second })
	fake := ours()
	fake.running, fake.refusedWaits = "false\n", 2
	c := &Cluster{kind: "kind", dir: t.TempDir(), run: fake.run}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.True(t, state.Started)
	assert.Zero(t, fake.refusedWaits)
}

func TestClusterEnsure_FailedStartShowsDockersReason(t *testing.T) {
	fake := ours()
	fake.running, fake.startErr = "false\n", errors.New("exit status 1")
	c := &Cluster{kind: "kind", dir: t.TempDir(), run: fake.run}

	_, out, err := c.Ensure(context.Background())

	assert.ErrorIs(t, err, ErrClusterStart)
	assert.Contains(t, out, "port is already allocated")
}

func TestCheckDisk_WarnsWhenLowOrUnreadable(t *testing.T) {
	df := func(availableKB string) string {
		return "Filesystem 1024-blocks Used Available Capacity Mounted on\noverlay 61202244 1000 " + availableKB + " 2% /\n"
	}
	tests := []struct {
		name   string
		out    string
		err    error
		status Status
	}{
		{"plenty of space", df("44363220"), nil, StatusPass},
		{"under 10 GB", df("5242880"), nil, StatusWarn},
		{"node not answering", "", errors.New("exit status 1"), StatusWarn},
		{"unexpected output", "df: /var: No such file", nil, StatusWarn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{run: func(context.Context, string, ...string) ([]byte, error) { return []byte(tt.out), tt.err }}

			r, _ := c.CheckDisk(context.Background())

			assert.Equal(t, tt.status, r.Status)
		})
	}
}
