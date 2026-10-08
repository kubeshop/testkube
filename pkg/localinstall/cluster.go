package localinstall

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	ClusterName       = "testkube"
	KubernetesVersion = "v1.37.0"

	// kind requires the digest; the tag alone may change.
	nodeImage = "kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5"

	// kind's storage writes here; mounting it keeps data on the host.
	nodeStorageDir = "/var/local-path-provisioner"

	// Mount destinations survive a stop; Docker never rewrites them.
	ownerMountDir = "/testkube/owner/"

	nodeName = ClusterName + "-control-plane"
)

var (
	ErrClusterNotOurs    = errors.New("a cluster with our name exists that this installer did not create")
	ErrClusterStart      = errors.New("the stopped cluster could not start")
	ErrClusterStorage    = errors.New("the cluster's storage could not be set up")
	ErrClusterInspect    = errors.New("the existing cluster could not be read")
	ErrClusterKubeconfig = errors.New("the cluster's kubeconfig could not be saved")
)

// Another program bound the port between our check and kind's.
type PortTakenError struct{ Port int }

func (e PortTakenError) Error() string {
	return fmt.Sprintf("port %d was taken before the cluster could use it", e.Port)
}

// Docker's wording, then Podman's.
var portTakenLine = regexp.MustCompile(`127\.0\.0\.1:(\d+).*(port is already allocated|address already in use)`)

func portTaken(out string) (int, bool) {
	m := portTakenLine.FindStringSubmatch(out)
	if m == nil {
		return 0, false
	}
	port, err := strconv.Atoi(m[1])
	return port, err == nil
}

// Browser-facing; Testkube's settings embed these exact addresses.
var clusterPorts = []struct {
	name          string
	host, inNodes int
}{
	{"dashboard", 8080, 30080},
	{"api", 8090, 30090},
	{"login", 5556, 30556},
	{"storage", 9000, 30900},
	{"ai", 9090, 30990},
}

type Ports map[string]int

func (p Ports) Moved() []string {
	var moved []string
	for _, cp := range clusterPorts {
		if p[cp.name] != cp.host {
			moved = append(moved, fmt.Sprintf("%s %d→%d", cp.name, cp.host, p[cp.name]))
		}
	}
	return moved
}

type ClusterState struct {
	Created bool
	Started bool
	Ports   Ports
}

// The API refuses connections for a few seconds after start.
var (
	startTimeout = 2 * time.Minute
	startRetry   = 2 * time.Second
)

type runFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

type Cluster struct {
	kind       string
	dir        string
	kubeconfig string
	run        runFunc
	portFree   func(port int) bool
}

func NewCluster() (*Cluster, error) {
	dir, err := ToolsDir()
	if err != nil {
		return nil, err
	}
	kubeconfig, err := KubeconfigPath()
	if err != nil {
		return nil, err
	}
	return &Cluster{kind: filepath.Join(dir, "kind"), dir: filepath.Dir(kubeconfig), kubeconfig: kubeconfig,
		run: runCombined, portFree: portFree}, nil
}

// Separate from ~/.kube/config, so the user's context never changes.
func KubeconfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".testkube", "kubeconfig"), nil
}

func (c *Cluster) Ensure(ctx context.Context) (ClusterState, string, error) {
	out, err := c.run(ctx, c.kind, "get", "clusters")
	if err != nil {
		return ClusterState{}, string(out), err
	}
	if hasLine(string(out), ClusterName) {
		return c.reuse(ctx)
	}

	busy := map[int]bool{}
	free := func(port int) bool { return !busy[port] && c.portFree(port) }
	ports := pickPorts(free)
	data := filepath.Join(c.dir, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		return ClusterState{}, "", err
	}
	// Pods write as their own users (postgres is uid 70 or 999).
	if err := os.Chmod(data, 0o777); err != nil {
		return ClusterState{}, "", err
	}
	// Saved first, so a Ctrl+C'd half-built node stays ours.
	owner := rand.Text()
	if err := os.WriteFile(c.ownerFile(), []byte(owner), 0o600); err != nil {
		return ClusterState{}, "", err
	}
	config := filepath.Join(c.dir, "kind.yaml")
	for attempt := 1; ; attempt++ {
		if err := os.WriteFile(config, []byte(kindConfig(ports, data, owner)), 0o644); err != nil {
			return ClusterState{}, "", err
		}
		out, err = c.run(ctx, c.kind, "create", "cluster", "--name", ClusterName, "--config", config,
			"--image", nodeImage, "--kubeconfig", c.kubeconfig, "--wait", "2m")
		if err == nil {
			break
		}
		port, taken := portTaken(string(out))
		if !taken {
			return ClusterState{Ports: ports}, string(out), err
		}
		if attempt == 2 {
			return ClusterState{Ports: ports}, string(out), PortTakenError{Port: port}
		}
		// kind removes the failed node, so the retry starts clean.
		busy[port] = true
		ports = pickPorts(free)
	}
	why, err := c.fixStorage(ctx)
	return ClusterState{Created: true, Ports: ports}, why, err
}

func (c *Cluster) reuse(ctx context.Context) (ClusterState, string, error) {
	var state ClusterState
	owner, err := os.ReadFile(c.ownerFile())
	if err != nil || len(owner) == 0 {
		return state, "", ErrClusterNotOurs
	}
	// Saved settings survive a stop: check before touching it.
	out, err := c.run(ctx, "docker", "inspect", "-f",
		"{{.State.Running}}\n{{json .HostConfig.PortBindings}}\n{{json .Mounts}}", nodeName)
	if err != nil {
		return state, string(out), ErrClusterInspect
	}
	running, ports, ok := ownedSettings(string(out), string(owner))
	if !ok {
		return state, "", ErrClusterNotOurs
	}
	state.Ports = ports
	if !running {
		if out, err = c.run(ctx, "docker", "start", nodeName); err != nil {
			return state, string(out), ErrClusterStart
		}
		state.Started = true
	}
	// Also catches a cluster left half-built by Ctrl+C.
	if out, err = c.waitReady(ctx); err != nil {
		return ClusterState{}, string(out), ErrClusterStart
	}
	if why, err := c.fixStorage(ctx); err != nil {
		return state, why, err
	}
	// Rewrites our kubeconfig in case it was deleted.
	if out, err = c.run(ctx, c.kind, "export", "kubeconfig", "--name", ClusterName, "--kubeconfig", c.kubeconfig); err != nil {
		return state, string(out), ErrClusterKubeconfig
	}
	return state, "", nil
}

// Ours means our saved mark and every browser port on 127.0.0.1.
func ownedSettings(inspect, owner string) (running bool, ports Ports, ok bool) {
	lines := strings.Split(strings.TrimSpace(inspect), "\n")
	if len(lines) != 3 {
		return false, nil, false
	}
	var bindings map[string][]struct{ HostIp, HostPort string }
	var mounts []struct{ Destination string }
	if json.Unmarshal([]byte(lines[1]), &bindings) != nil || json.Unmarshal([]byte(lines[2]), &mounts) != nil {
		return false, nil, false
	}
	if !slices.ContainsFunc(mounts, func(m struct{ Destination string }) bool { return m.Destination == ownerMountDir+owner }) {
		return false, nil, false
	}
	ports = Ports{}
	for _, cp := range clusterPorts {
		b := bindings[fmt.Sprintf("%d/tcp", cp.inNodes)]
		if len(b) != 1 || b[0].HostIp != "127.0.0.1" {
			return false, nil, false
		}
		host, err := strconv.Atoi(b[0].HostPort)
		if err != nil {
			return false, nil, false
		}
		ports[cp.name] = host
	}
	return lines[0] == "true", ports, true
}

func (c *Cluster) ownerFile() string {
	return filepath.Join(c.dir, "cluster-id")
}

// kind's default folders are random; a rebuilt cluster loses data.
const storagePathPattern = "{{ .PVC.Namespace }}/{{ .PVC.Name }}/data"

// Retain: helm uninstall must not wipe the user's data folder.
const storageClass = `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: rancher.io/local-path
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Retain
parameters:
  pathPattern: "` + storagePathPattern + `"
`

func (c *Cluster) fixStorage(ctx context.Context) (string, error) {
	out, err := c.nodeKubectl(ctx, "get", "storageclass", "standard", "-o", "jsonpath={.parameters.pathPattern}")
	if err == nil && string(out) == storagePathPattern {
		return "", nil
	}
	// Parameters can't change in place, so the class is replaced.
	if out, err = c.nodeKubectl(ctx, "delete", "storageclass", "standard", "--ignore-not-found"); err != nil {
		return string(out), ErrClusterStorage
	}
	out, err = c.run(ctx, "docker", "exec", nodeName, "sh", "-c",
		`printf '%s' "$1" | kubectl --kubeconfig=/etc/kubernetes/admin.conf apply -f -`, "_", storageClass)
	if err != nil {
		return string(out), ErrClusterStorage
	}
	return "", nil
}

func (c *Cluster) nodeKubectl(ctx context.Context, args ...string) ([]byte, error) {
	return c.run(ctx, "docker", append([]string{"exec", nodeName, "kubectl", "--kubeconfig=/etc/kubernetes/admin.conf"}, args...)...)
}

func (c *Cluster) waitReady(ctx context.Context) ([]byte, error) {
	deadline := time.Now().Add(startTimeout)
	for {
		out, err := c.run(ctx, "docker", "exec", nodeName, "kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
			"wait", "--for=condition=Ready", "nodes", "--all", "--timeout=10s")
		if err == nil || time.Now().After(deadline) {
			return out, err
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case <-time.After(startRetry):
		}
	}
}

// Docker Desktop hides its disk from the host; the node sees it.
func (c *Cluster) CheckDisk(ctx context.Context) (r Result, freeGB float64) {
	out, err := c.run(ctx, "docker", "exec", nodeName, "df", "-Pk", "/var")
	free, ok := parseDfAvailable(string(out))
	if err != nil || !ok {
		return Result{Name: "disk", Status: StatusWarn, Detail: "could not read free space"}, 0
	}
	return minimumResult("disk", free >= minFreeDisk, formatGB(free)+" free", "needs "+formatGB(minFreeDisk)), roundGB(free)
}

// df -P prints a header, then: filesystem, blocks, used, available.
func parseDfAvailable(out string) (uint64, bool) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return 0, false
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 4 {
		return 0, false
	}
	kb, err := strconv.ParseUint(fields[3], 10, 64)
	return kb * 1024, err == nil
}

func pickPorts(free func(int) bool) Ports {
	ports, taken := Ports{}, map[int]bool{}
	for _, cp := range clusterPorts {
		port := cp.host
		for taken[port] || !free(port) {
			port++
		}
		taken[port] = true
		ports[cp.name] = port
	}
	return ports
}

func portFree(port int) bool {
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// 127.0.0.1 only: a trial must not be reachable from the network.
func kindConfig(ports Ports, dataDir, owner string) string {
	var b strings.Builder
	b.WriteString("kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nnodes:\n- role: control-plane\n  extraPortMappings:\n")
	for _, cp := range clusterPorts {
		fmt.Fprintf(&b, "  - containerPort: %d\n    hostPort: %d\n    listenAddress: 127.0.0.1\n", cp.inNodes, ports[cp.name])
	}
	fmt.Fprintf(&b, "  extraMounts:\n  - hostPath: %q\n    containerPath: %s\n", dataDir, nodeStorageDir)
	fmt.Fprintf(&b, "  - hostPath: %q\n    containerPath: %s\n    readOnly: true\n", dataDir, ownerMountDir+owner)
	return b.String()
}

// Exact match: "testkube-dev" is someone else's cluster.
func hasLine(out, want string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

func runCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.Bytes(), err
}
