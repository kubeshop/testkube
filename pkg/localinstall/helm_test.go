package localinstall

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHelm_AlwaysTargetsOurCluster(t *testing.T) {
	var got []string
	h := &Helm{bin: "helm", home: "home", kubeconfig: "our-kubeconfig",
		exec: func(_ context.Context, _ []string, _ string, args ...string) ([]byte, error) {
			got = args
			return nil, nil
		}}

	_, _ = h.Run(context.Background(), "list", "-a")

	assert.Equal(t, []string{"list", "-a", "--kubeconfig", "our-kubeconfig", "--kube-context", "kind-testkube"}, got)
}

func TestHelmEnv_IgnoresTheUsersHelmKubeAndDockerSetup(t *testing.T) {
	home := filepath.Join("h", "helm")
	env := helmEnv([]string{
		"PATH=/usr/bin", "HTTPS_PROXY=http://proxy:3128",
		"HELM_KUBECONTEXT=prod", "HELM_REPOSITORY_CONFIG=/user/repos.yaml",
		"KUBECONFIG=/user/.kube/config", "KUBECACHEDIR=/user/.kube/cache", "DOCKER_CONFIG=/user/.docker",
	}, home)

	assert.Subset(t, env, []string{
		"PATH=/usr/bin", "HTTPS_PROXY=http://proxy:3128",
		"HELM_CONFIG_HOME=" + filepath.Join(home, "config"),
		"HELM_CACHE_HOME=" + filepath.Join(home, "cache"),
		"HELM_DATA_HOME=" + filepath.Join(home, "data"),
		"KUBECACHEDIR=" + filepath.Join(home, "kube"),
		"DOCKER_CONFIG=" + filepath.Join(home, "docker"),
	})
	for _, kv := range env {
		assert.False(t, strings.Contains(kv, "/user/") || strings.HasPrefix(kv, "HELM_KUBECONTEXT"), kv)
	}
}

func TestRunGentle_CtrlCLetsHelmCleanUp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no SIGINT for child processes")
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)

	out, _ := runGentle(ctx, nil, "sh", "-c", `trap 'echo marked failed; exit 1' INT; sleep 5 >/dev/null 2>&1 & wait`)

	assert.Contains(t, string(out), "marked failed")
}
