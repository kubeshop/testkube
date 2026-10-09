package localinstall

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
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
	clusters, running, bindings, mounts string
	storagePattern, createOut           string
	applyErr, inspectErr, exportErr     error
	startErr                            error
	refusedWaits, createFails           int
	calls                               []string
}

func (f *kindFake) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	switch {
	case name != "docker" && args[0] == "get":
		return []byte(f.clusters), nil
	case name != "docker" && args[0] == "create" && f.createFails > 0:
		f.createFails--
		return []byte(f.createOut), errExit
	case name != "docker" && args[0] == "export":
		return nil, f.exportErr
	case args[0] == "inspect":
		return []byte(f.running + "\n" + f.bindings + "\n" + f.mounts + "\n"), f.inspectErr
	case args[0] == "start":
		return []byte("Bind for 127.0.0.1:9000 failed: port is already allocated"), f.startErr
	case args[0] == "exec" && f.refusedWaits > 0:
		f.refusedWaits--
		return []byte("The connection to the server testkube-control-plane:6443 was refused"), errors.New("exit status 1")
	case args[0] == "exec" && slices.Contains(args, "get") && slices.Contains(args, "storageclass"):
		return []byte(f.storagePattern), nil
	case args[0] == "exec" && args[2] == "sh":
		return []byte("error: storageclasses.storage.k8s.io is forbidden"), f.applyErr
	}
	return nil, nil
}

func ours(t *testing.T, dir string) *kindFake {
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cluster-id"), []byte("OURMARK"), 0o600))
	mounts := `[{"Destination":"/lib/modules"},{"Destination":"/var/local-path-provisioner"},{"Destination":"/testkube/owner/OURMARK"}]`
	return &kindFake{clusters: "testkube\n", running: "true", bindings: ourBindings, mounts: mounts, storagePattern: storagePathPattern}
}

func allFree(int) bool { return true }

func TestClusterEnsure_ReusesOursOnly(t *testing.T) {
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "kubeconfig")
	create := "kind create cluster --name testkube --config " + filepath.Join(dir, "kind.yaml") +
		" --image " + nodeImage + " --kubeconfig " + kubeconfig + " --wait 2m"
	storage := "docker exec testkube-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf get storageclass standard -o jsonpath={.parameters.pathPattern}"
	tests := []struct {
		name        string
		clusters    string
		wantCreated bool
		wantCalls   []string
	}{
		{"our cluster is reused, not recreated", "kind\ntestkube\n", false, []string{
			"kind get clusters",
			"docker inspect -f {{.State.Running}}\n{{json .HostConfig.PortBindings}}\n{{json .Mounts}} testkube-control-plane",
			"docker exec testkube-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf wait --for=condition=Ready nodes --all --timeout=10s",
			storage,
			"kind export kubeconfig --name testkube --kubeconfig " + kubeconfig,
		}},
		{"a similar name is someone else's", "testkube-dev\n", true, []string{"kind get clusters", create, storage}},
		{"no clusters at all", "No kind clusters found.\n", true, []string{"kind get clusters", create, storage}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := ours(t, dir)
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
	c := &Cluster{kind: "kind", dir: dir, run: ours(t, dir).run}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 9001, state.Ports["storage"])
	assert.Equal(t, []string{"storage 9000→9001"}, state.Ports.Moved())
}

func TestClusterEnsure_SomeoneElsesClusterIsNeverStarted(t *testing.T) {
	tests := map[string]func(f *kindFake, dir string){
		"ports open to the network": func(f *kindFake, _ string) { f.bindings = strings.ReplaceAll(ourBindings, "127.0.0.1", "0.0.0.0") },
		"no browser ports":          func(f *kindFake, _ string) { f.bindings = `{"6443/tcp":[{"HostIp":"127.0.0.1","HostPort":"51412"}]}` },
		"no owner mark":             func(f *kindFake, _ string) { f.mounts = `[{"Destination":"/var/local-path-provisioner"}]` },
		"another install's mark": func(_ *kindFake, dir string) {
			_ = os.WriteFile(filepath.Join(dir, "cluster-id"), []byte("OTHER"), 0o600)
		},
		"our saved mark is gone":  func(_ *kindFake, dir string) { _ = os.Remove(filepath.Join(dir, "cluster-id")) },
		"our saved mark is empty": func(_ *kindFake, dir string) { _ = os.WriteFile(filepath.Join(dir, "cluster-id"), nil, 0o600) },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			fake := ours(t, dir)
			fake.running = "false"
			change(fake, dir)
			c := &Cluster{kind: "kind", dir: dir, run: fake.run}

			_, _, err := c.Ensure(context.Background())

			assert.ErrorIs(t, err, ErrClusterNotOurs)
			assert.NotContains(t, fake.calls, "docker start testkube-control-plane")
		})
	}
}

var errExit = errors.New("exit status 1")

func TestClusterEnsure_TakenPortIsNamed(t *testing.T) {
	tests := map[string]struct {
		out  string
		want error
	}{
		"docker":          {"Error response from daemon: Bind for 127.0.0.1:9001 failed: port is already allocated", PortTakenError{Port: 9001}},
		"podman":          {"Error: rootlessport listen tcp 127.0.0.1:8090: bind: address already in use", PortTakenError{Port: 8090}},
		"another failure": {"ERROR: failed to create cluster: no space left on device", errExit},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fake := &kindFake{createOut: tt.out, createFails: 2}
			c := &Cluster{kind: "kind", dir: t.TempDir(), portFree: allFree, run: fake.run}

			_, _, err := c.Ensure(context.Background())

			assert.Equal(t, tt.want, err)
		})
	}
}

func TestClusterEnsure_PortTakenDuringCreateMovesToAFreeOne(t *testing.T) {
	fake := &kindFake{createOut: "Bind for 127.0.0.1:9000 failed: port is already allocated", createFails: 1}
	c := &Cluster{kind: "kind", dir: t.TempDir(), portFree: allFree, run: fake.run}

	state, _, err := c.Ensure(context.Background())

	require.NoError(t, err)
	assert.True(t, state.Created)
	assert.Equal(t, []string{"storage 9000→9001"}, state.Ports.Moved())
}

func TestClusterEnsure_ReuseFailuresAreNamed(t *testing.T) {
	tests := map[string]struct {
		change func(f *kindFake)
		want   error
	}{
		"cluster can't be read":     {func(f *kindFake) { f.inspectErr = errExit }, ErrClusterInspect},
		"kubeconfig can't be saved": {func(f *kindFake) { f.exportErr = errExit }, ErrClusterKubeconfig},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			fake := ours(t, dir)
			tt.change(fake)
			c := &Cluster{kind: "kind", dir: dir, run: fake.run}

			_, _, err := c.Ensure(context.Background())

			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestPickPorts_MovesBusyPortsWithoutCollisions(t *testing.T) {
	busy := map[int]bool{8080: true, 8081: true, 8082: true, 8083: true, 8084: true,
		8085: true, 8086: true, 8087: true, 8088: true, 8089: true, 9000: true}

	ports := pickPorts(func(p int) bool { return !busy[p] })

	assert.Equal(t, Ports{"dashboard": 8090, "api": 8091, "login": 5556, "storage": 9001, "ai": 9090}, ports)
}

// A dev server on [::] shared its port; localhost hit it.
func TestPortFree_AnotherAppAnsweringLocalhostIsBusy(t *testing.T) {
	for _, addr := range []string{"[::1]:0", "[::]:0", "0.0.0.0:0"} {
		t.Run(addr, func(t *testing.T) {
			l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
			if err != nil {
				t.Skip("no IPv6 here:", err)
			}
			defer l.Close()

			assert.False(t, portFree(l.Addr().(*net.TCPAddr).Port))
		})
	}
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

func TestClusterEnsure_NewClusterCarriesTheMarkWeSaved(t *testing.T) {
	dir := t.TempDir()
	c := &Cluster{kind: "kind", dir: dir, portFree: allFree, run: (&kindFake{}).run}

	_, _, err := c.Ensure(context.Background())
	require.NoError(t, err)

	owner, err := os.ReadFile(filepath.Join(dir, "cluster-id"))
	require.NoError(t, err)
	require.NotEmpty(t, owner)
	config, _ := os.ReadFile(filepath.Join(dir, "kind.yaml"))
	assert.Contains(t, string(config), "containerPath: /testkube/owner/"+string(owner)+"\n    readOnly: true")
}

func TestClusterEnsure_DataFoldersSurviveARebuild(t *testing.T) {
	tests := map[string]*kindFake{
		"new cluster":                   {},
		"cluster from an older install": nil,
	}
	for name, fake := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if fake == nil {
				fake = ours(t, dir)
				fake.storagePattern = ""
			}
			c := &Cluster{kind: "kind", dir: dir, portFree: allFree, run: fake.run}

			_, _, err := c.Ensure(context.Background())
			require.NoError(t, err)

			i := slices.IndexFunc(fake.calls, func(c string) bool { return strings.Contains(c, "apply -f -") })
			require.NotEqual(t, -1, i)
			apply := fake.calls[i]
			assert.Contains(t, fake.calls, "docker exec testkube-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf delete storageclass standard --ignore-not-found")
			assert.Contains(t, apply, `pathPattern: "`+storagePathPattern+`"`)
			assert.Contains(t, apply, "reclaimPolicy: Retain")
		})
	}
}

func TestClusterEnsure_StorageFailureIsNamed(t *testing.T) {
	fake := &kindFake{applyErr: errors.New("exit status 1")}
	c := &Cluster{kind: "kind", dir: t.TempDir(), portFree: allFree, run: fake.run}

	_, out, err := c.Ensure(context.Background())

	assert.ErrorIs(t, err, ErrClusterStorage)
	assert.Contains(t, out, "forbidden")
}

func TestClusterEnsure_StoppedClusterIsStartedBeforeReading(t *testing.T) {
	dir := t.TempDir()
	fake := ours(t, dir)
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
	fake := ours(t, dir)
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
	fake := ours(t, dir)
	fake.refusedWaits = 1 << 30
	c := &Cluster{kind: "kind", dir: dir, run: fake.run}

	_, out, err := c.Ensure(context.Background())

	assert.ErrorIs(t, err, ErrClusterStart)
	assert.Contains(t, out, "refused")
}

func TestClusterEnsure_FailedStartShowsDockersReason(t *testing.T) {
	dir := t.TempDir()
	fake := ours(t, dir)
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
