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

// Storage moved off its default, as a busy 9000 would cause.
const ourBindings = `{"30080/tcp":[{"HostIp":"127.0.0.1","HostPort":"8080"}],` +
	`"30090/tcp":[{"HostIp":"127.0.0.1","HostPort":"8090"}],"30556/tcp":[{"HostIp":"127.0.0.1","HostPort":"5556"}],` +
	`"30900/tcp":[{"HostIp":"127.0.0.1","HostPort":"9001"}],"30990/tcp":[{"HostIp":"127.0.0.1","HostPort":"9090"}],` +
	`"6443/tcp":[{"HostIp":"127.0.0.1","HostPort":"51412"}]}`

type kindFake struct {
	clusters, running, bindings, binds, labels string
	startErr                                   error
	refusedWaits                               int
	calls                                      []string
}

func (f *kindFake) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	switch {
	case name != "docker" && args[0] == "get":
		return []byte(f.clusters), nil
	case args[0] == "inspect":
		return []byte(f.running + "\n" + f.bindings + "\n" + f.binds + "\n" + f.labels + "\n"), nil
	case args[0] == "start":
		return []byte("Bind for 127.0.0.1:9000 failed: port is already allocated"), f.startErr
	case args[0] == "exec" && f.refusedWaits > 0:
		f.refusedWaits--
		return []byte("The connection to the server testkube-control-plane:6443 was refused"), errors.New("exit status 1")
	}
	return nil, nil
}

// A healthy, running cluster of ours; Docker Desktop prefixes the source.
func ours(dir string) *kindFake {
	binds := fmt.Sprintf(`["/lib/modules:/lib/modules:ro","/host_mnt%s:/var/local-path-provisioner"]`, filepath.Join(dir, "data"))
	return &kindFake{clusters: "testkube\n", running: "true", bindings: ourBindings, binds: binds, labels: "{}"}
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
			"docker inspect -f {{.State.Running}}\n{{json .HostConfig.PortBindings}}\n{{json .HostConfig.Binds}}\n{{json .Config.Labels}} testkube-control-plane",
			"docker exec testkube-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf wait --for=condition=Ready nodes --all --timeout=10s",
			"kind export kubeconfig --name testkube --kubeconfig " + kubeconfig,
		}},
		{"a similar name is someone else's", "testkube-dev\n", true, []string{"kind get clusters", create}},
		{"no clusters at all", "No kind clusters found.\n", true, []string{"kind get clusters", create}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := ours(dir)
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
	dir := t.TempDir()
	c := &Cluster{kind: "kind", dir: dir, run: ours(dir).run}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 9001, state.Ports["storage"])
	assert.Equal(t, []string{"storage 9000→9001"}, state.Ports.Moved())
}

func TestClusterEnsure_SomeoneElsesClusterIsNeverStarted(t *testing.T) {
	dir := t.TempDir()
	tests := map[string]func(f *kindFake){
		"ports open to the network": func(f *kindFake) { f.bindings = strings.ReplaceAll(ourBindings, "127.0.0.1", "0.0.0.0") },
		"no browser ports":          func(f *kindFake) { f.bindings = `{"6443/tcp":[{"HostIp":"127.0.0.1","HostPort":"51412"}]}` },
		"data in another folder":    func(f *kindFake) { f.binds = `["/home/other/data:/var/local-path-provisioner"]` },
		"no data folder":            func(f *kindFake) { f.binds = `["/lib/modules:/lib/modules:ro"]` },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			fake := ours(dir)
			fake.running = "false"
			change(fake)
			c := &Cluster{kind: "kind", dir: dir, run: fake.run}

			state, _, err := c.Ensure(context.Background())

			assert.ErrorIs(t, err, ErrClusterNotOurs)
			assert.Len(t, fake.calls, 2, "only listed and inspected, never started")
			if name == "data in another folder" {
				assert.Equal(t, "/home", state.DataSourceRoot, "first folder only, never the full path")
			}
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
	dir := t.TempDir()
	fake := ours(dir)
	fake.running = "false"
	c := &Cluster{kind: "kind", dir: dir, run: fake.run}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.True(t, state.Started)
	assert.Equal(t, "docker start testkube-control-plane", fake.calls[2])
	assert.Contains(t, fake.calls[3], "wait --for=condition=Ready")
}

func TestClusterEnsure_WaitsWhileTheAPIIsStillStarting(t *testing.T) {
	startRetry = time.Millisecond
	t.Cleanup(func() { startRetry = 2 * time.Second })
	dir := t.TempDir()
	fake := ours(dir)
	fake.running, fake.refusedWaits = "false", 2
	c := &Cluster{kind: "kind", dir: dir, run: fake.run}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.True(t, state.Started)
	assert.Zero(t, fake.refusedWaits)
}

func TestClusterEnsure_RunningButHalfBuiltClusterIsNotReused(t *testing.T) {
	startTimeout, startRetry = 5*time.Millisecond, time.Millisecond
	t.Cleanup(func() { startTimeout, startRetry = 2*time.Minute, 2*time.Second })
	dir := t.TempDir()
	fake := ours(dir)
	fake.refusedWaits = 1 << 30
	c := &Cluster{kind: "kind", dir: dir, run: fake.run}

	_, out, err := c.Ensure(context.Background())

	assert.ErrorIs(t, err, ErrClusterStart)
	assert.Contains(t, out, "refused")
}

func TestClusterEnsure_FailedStartShowsDockersReason(t *testing.T) {
	dir := t.TempDir()
	fake := ours(dir)
	fake.running, fake.startErr = "false", errors.New("exit status 1")
	c := &Cluster{kind: "kind", dir: dir, run: fake.run}

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

func TestMountsData_RecognizesOurFolderOnEveryDockerSetup(t *testing.T) {
	const data = "/home/u/.testkube/data"
	wsl := `/run/desktop/mnt/host/wsl/docker-desktop-bind-mounts/Ubuntu/9f2c1e:/var/local-path-provisioner`
	tests := []struct {
		name   string
		bind   string
		labels map[string]string
		ours   bool
	}{
		{"linux and colima keep the path", data + ":/var/local-path-provisioner", nil, true},
		{"docker desktop on mac prefixes it", "/host_mnt" + data + ":/var/local-path-provisioner", nil, true},
		{"podman adds options", data + ":/var/local-path-provisioner:rw,rprivate,rbind", nil, true},
		{"docker desktop on wsl, label is ours", wsl, map[string]string{"desktop.docker.io/binds/1/Source": data}, true},
		{"docker desktop on wsl, label is another folder", wsl, map[string]string{"desktop.docker.io/binds/1/Source": "/home/other/data"}, false},
		{"rancher desktop on wsl hides the path", "/mnt/wsl/rancher-desktop/run/docker-mounts/1b2c:/var/local-path-provisioner", nil, true},
		{"another folder", "/home/other/data:/var/local-path-provisioner", nil, false},
		{"a folder that only ends like ours", "/tmp" + data + ":/var/local-path-provisioner", nil, false},
		{"a longer destination is not ours", data + ":/var/local-path-provisioner-old", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ours, _ := mountsData([]string{tt.bind}, tt.labels, data)
			assert.Equal(t, tt.ours, ours)
		})
	}
}
