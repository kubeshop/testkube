package localinstall

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ClusterName       = "testkube"
	KubernetesVersion = "v1.37.0"

	// kind requires the digest; the tag alone may change.
	nodeImage = "kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5"
)

type runFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

type Cluster struct {
	kind       string
	kubeconfig string
	run        runFunc
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
	return &Cluster{kind: filepath.Join(dir, "kind"), kubeconfig: kubeconfig, run: runCombined}, nil
}

// Separate from ~/.kube/config, so the user's context never changes.
func KubeconfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".testkube", "kubeconfig"), nil
}

func (c *Cluster) Ensure(ctx context.Context) (created bool, output string, err error) {
	out, err := c.run(ctx, c.kind, "get", "clusters")
	if err != nil {
		return false, string(out), err
	}
	if hasLine(string(out), ClusterName) {
		// Rewrites our kubeconfig in case it was deleted.
		out, err = c.run(ctx, c.kind, "export", "kubeconfig", "--name", ClusterName, "--kubeconfig", c.kubeconfig)
		return false, string(out), err
	}
	out, err = c.run(ctx, c.kind, "create", "cluster", "--name", ClusterName,
		"--image", nodeImage, "--kubeconfig", c.kubeconfig, "--wait", "2m")
	return err == nil, string(out), err
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
