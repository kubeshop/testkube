package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/localinstall"
)

func TestResourceFix_PointsAtEachDockerAppsOwnSetting(t *testing.T) {
	tests := map[string]struct {
		docker localinstall.DockerResources
		cpu    bool
		want   string
	}{
		"Docker Desktop memory": {localinstall.DockerResources{Engine: "Docker Desktop", Memory: 4 << 30}, false, "Settings > Resources, set Memory to 6 GB"},
		"Colima CPU":            {localinstall.DockerResources{Engine: "colima", CPUs: 2}, true, "colima start --cpu 4"},
		"Linux memory":          {localinstall.DockerResources{Engine: "Ubuntu 24.04 LTS", Memory: 4 << 30}, false, "Free up memory"},
		"Docker unreadable":     {localinstall.DockerResources{}, false, "Give Docker 6 GB or more"},
		"enough on paper":       {localinstall.DockerResources{Engine: "Docker Desktop", Memory: 8 << 30}, false, "isn't enough free memory in the cluster"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Contains(t, resourceFix(tt.cpu, tt.docker), tt.want)
		})
	}
}

func TestHomeRelative_ShortensOnlyPathsInsideHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	inside := filepath.Join(home, ".testkube", "logs", "install-1.txt")

	assert.Equal(t, "~"+string(os.PathSeparator)+filepath.Join(".testkube", "logs", "install-1.txt"), homeRelative(inside))
	assert.Equal(t, home+"2/x.txt", homeRelative(home+"2/x.txt"), "a sibling folder isn't home")
}

func TestDownloadedText_HiddenOnRerunsAndHonestPastTheEstimate(t *testing.T) {
	tests := map[string]struct {
		bytes int64
		want  string
	}{
		"rerun, images cached": {bytes: 999_999, want: ""},
		"mid download":         {bytes: 640_000_000, want: "640 MB of ~1.2 GB · "},
		"past the estimate":    {bytes: 1_210_000_000, want: "1.2 GB downloaded · "},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, downloadedText(tt.bytes))
		})
	}
}

func TestFailedChecks_NamesOnlyTheFailures(t *testing.T) {
	results := []localinstall.Result{
		{Name: "kubectl", Status: localinstall.StatusWarn, Detail: "not found"},
		{Name: "docker", Status: localinstall.StatusFail, Detail: "not reachable"},
		{Name: "helm", Status: localinstall.StatusPass, Detail: "ready"},
	}

	assert.Equal(t, "docker not reachable", failedChecks(results))
}

// Wrong buckets would send the PM chasing the wrong problem.
func TestFailureCategory_SortsByWhatTheUserMustFix(t *testing.T) {
	tests := map[string]struct {
		stage, reason, want string
	}{
		"bad key":               {"license", "not valid", "license"},
		"license server down":   {"license", "cannot reach license.testkube.io", "network"},
		"docker missing":        {"tools", "docker not found", "docker"},
		"tool download":         {"tool_install", "helm download", "network"},
		"tools folder unusable": {"tool_install", "kind save", "machine_resources"},
		"cluster storage":       {"cluster", "storage", "machine_resources"},
		"cluster create":        {"cluster", "create", "docker"},
		"too little memory":     {"testkube", "stuck no_memory", "machine_resources"},
		"docker hub limit":      {"testkube", "stuck rate_limit", "network"},
		"service crashloops":    {"testkube", "stuck crashloop", "install_stuck"},
		"timeout":               {"testkube", "timeout", "install_stuck"},
		"unknown stage":         {"later", "x", "other"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, failureCategory(tt.stage, tt.reason))
		})
	}
}
