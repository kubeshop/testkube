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
)

const (
	ClusterName       = "testkube"
	KubernetesVersion = "v1.37.0"

	// kind requires the digest; the tag alone may change.
	nodeImage = "kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5"
)

var ErrClusterWithoutPorts = errors.New("the existing cluster has no Testkube port mappings")

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
	Ports   Ports
}

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
		// Rewrites our kubeconfig in case it was deleted.
		if out, err = c.run(ctx, c.kind, "export", "kubeconfig", "--name", ClusterName, "--kubeconfig", c.kubeconfig); err != nil {
			return ClusterState{}, string(out), err
		}
		// Mappings are fixed at creation, so read what it got.
		if out, err = c.run(ctx, "docker", "port", ClusterName+"-control-plane"); err != nil {
			return ClusterState{}, string(out), err
		}
		ports := parsePorts(string(out))
		if len(ports) != len(clusterPorts) {
			return ClusterState{}, "", ErrClusterWithoutPorts
		}
		return ClusterState{Ports: ports}, "", nil
	}

	ports := pickPorts(c.portFree)
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return ClusterState{}, "", err
	}
	config := filepath.Join(c.dir, "kind.yaml")
	if err := os.WriteFile(config, []byte(kindConfig(ports)), 0o644); err != nil {
		return ClusterState{}, "", err
	}
	out, err = c.run(ctx, c.kind, "create", "cluster", "--name", ClusterName, "--config", config,
		"--image", nodeImage, "--kubeconfig", c.kubeconfig, "--wait", "2m")
	return ClusterState{Created: err == nil, Ports: ports}, string(out), err
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
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// 127.0.0.1 only: a trial must not be reachable from the network.
func kindConfig(ports Ports) string {
	var b strings.Builder
	b.WriteString("kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nnodes:\n- role: control-plane\n  extraPortMappings:\n")
	for _, cp := range clusterPorts {
		fmt.Fprintf(&b, "  - containerPort: %d\n    hostPort: %d\n    listenAddress: 127.0.0.1\n", cp.inNodes, ports[cp.name])
	}
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
