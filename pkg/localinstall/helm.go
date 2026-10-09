package localinstall

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type execFunc func(ctx context.Context, env []string, name string, args ...string) ([]byte, error)

// Our helm, cluster and folders; the user's setup stays untouched.
type Helm struct {
	bin        string
	home       string
	kubeconfig string
	exec       execFunc
}

func NewHelm() (*Helm, error) {
	bin, err := ToolsDir()
	if err != nil {
		return nil, err
	}
	kubeconfig, err := KubeconfigPath()
	if err != nil {
		return nil, err
	}
	return &Helm{bin: filepath.Join(bin, "helm"), home: filepath.Join(filepath.Dir(kubeconfig), "helm"),
		kubeconfig: kubeconfig, exec: runGentle}, nil
}

func (h *Helm) Run(ctx context.Context, args ...string) ([]byte, error) {
	args = append(args, "--kubeconfig", h.kubeconfig, "--kube-context", "kind-"+ClusterName)
	return h.exec(ctx, helmEnv(os.Environ(), h.home), h.bin, args...)
}

// The user's HELM_* or KUBECONFIG would redirect helm.
func helmEnv(parent []string, home string) []string {
	env := make([]string, 0, len(parent)+5)
	for _, kv := range parent {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "HELM_") || key == "KUBECONFIG" || key == "KUBECACHEDIR" || key == "DOCKER_CONFIG" {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"HELM_CONFIG_HOME="+filepath.Join(home, "config"),
		"HELM_CACHE_HOME="+filepath.Join(home, "cache"),
		"HELM_DATA_HOME="+filepath.Join(home, "data"),
		// Otherwise helm caches into the user's ~/.kube/cache.
		"KUBECACHEDIR="+filepath.Join(home, "kube"),
		// Charts are public; the user's Docker logins stay unread.
		"DOCKER_CONFIG="+filepath.Join(home, "docker"),
	)
}

// SIGKILL leaves the release stuck; SIGINT lets helm clean up.
func runGentle(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = &out, &out
	cmd.Cancel = func() error {
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = 10 * time.Second
	err := cmd.Run()
	return out.Bytes(), err
}
