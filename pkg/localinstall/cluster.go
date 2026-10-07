package localinstall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	nodeName = ClusterName + "-control-plane"
)

var (
	ErrClusterOutdated = errors.New("the existing cluster lacks Testkube ports or data folder")
	ErrClusterStart    = errors.New("the stopped cluster could not start")
)

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

	ports := pickPorts(c.portFree)
	data := filepath.Join(c.dir, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		return ClusterState{}, "", err
	}
	// Pods write as their own users (postgres is uid 70 or 999).
	if err := os.Chmod(data, 0o777); err != nil {
		return ClusterState{}, "", err
	}
	config := filepath.Join(c.dir, "kind.yaml")
	if err := os.WriteFile(config, []byte(kindConfig(ports, data)), 0o644); err != nil {
		return ClusterState{}, "", err
	}
	out, err = c.run(ctx, c.kind, "create", "cluster", "--name", ClusterName, "--config", config,
		"--image", nodeImage, "--kubeconfig", c.kubeconfig, "--wait", "2m")
	return ClusterState{Created: err == nil, Ports: ports}, string(out), err
}

func (c *Cluster) reuse(ctx context.Context) (ClusterState, string, error) {
	var state ClusterState
	out, err := c.run(ctx, "docker", "inspect", "-f", "{{.State.Running}}", nodeName)
	if err != nil {
		return state, string(out), err
	}
	// A stopped node has no port bindings to read; start it first.
	if strings.TrimSpace(string(out)) != "true" {
		if out, err = c.run(ctx, "docker", "start", nodeName); err != nil {
			return state, string(out), ErrClusterStart
		}
		if out, err = c.waitReady(ctx); err != nil {
			return state, string(out), ErrClusterStart
		}
		state.Started = true
	}
	// Rewrites our kubeconfig in case it was deleted.
	if out, err = c.run(ctx, c.kind, "export", "kubeconfig", "--name", ClusterName, "--kubeconfig", c.kubeconfig); err != nil {
		return state, string(out), err
	}
	// Mappings are fixed at creation, so read what it got.
	if out, err = c.run(ctx, "docker", "port", nodeName); err != nil {
		return state, string(out), err
	}
	state.Ports = parsePorts(string(out))
	if out, err = c.run(ctx, "docker", "inspect", "-f", "{{range .Mounts}}{{.Destination}}\n{{end}}", nodeName); err != nil {
		return state, string(out), err
	}
	if len(state.Ports) != len(clusterPorts) || !hasLine(string(out), nodeStorageDir) {
		return ClusterState{}, "", ErrClusterOutdated
	}
	return state, "", nil
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
func kindConfig(ports Ports, dataDir string) string {
	var b strings.Builder
	b.WriteString("kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nnodes:\n- role: control-plane\n  extraPortMappings:\n")
	for _, cp := range clusterPorts {
		fmt.Fprintf(&b, "  - containerPort: %d\n    hostPort: %d\n    listenAddress: 127.0.0.1\n", cp.inNodes, ports[cp.name])
	}
	fmt.Fprintf(&b, "  extraMounts:\n  - hostPath: %q\n    containerPath: %s\n", dataDir, nodeStorageDir)
	return b.String()
}

// Lines look like "30080/tcp -> 127.0.0.1:8080".
var portLine = regexp.MustCompile(`^(\d+)/tcp -> [^:]+:(\d+)$`)

func parsePorts(out string) Ports {
	ports := Ports{}
	for _, line := range strings.Split(out, "\n") {
		m := portLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		inNodes, _ := strconv.Atoi(m[1])
		host, _ := strconv.Atoi(m[2])
		for _, cp := range clusterPorts {
			if cp.inNodes == inNodes {
				ports[cp.name] = host
			}
		}
	}
	return ports
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
