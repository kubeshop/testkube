package localinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeDocker struct {
	info dockerInfo
	err  error
}

func (f fakeDocker) Info(context.Context) (dockerInfo, error) {
	return f.info, f.err
}

type fakeHost struct {
	missing     map[string]bool
	free        uint64
	diskVisible bool
	diskErr     error
	blocked     map[string]bool
}

func (f fakeHost) LookPath(name string) (string, error) {
	if f.missing[name] {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}

func (f fakeHost) FreeDiskAt(string) (uint64, bool, error) {
	return f.free, f.diskVisible, f.diskErr
}

func (f fakeHost) ToolVersion(context.Context, string, ...string) string {
	return "v1.0.0"
}

func (f fakeHost) Reachable(_ context.Context, url string) bool {
	return !f.blocked[url]
}

func (f fakeHost) OSVersion() string {
	return "darwin 15.1"
}

func TestChecker_FactsReportRealValuesOnly(t *testing.T) {
	t.Run("reachable docker reports measured values", func(t *testing.T) {
		checker := &Checker{
			host: fakeHost{free: 100 * gigabyte, diskVisible: true},
			docker: fakeDocker{info: dockerInfo{NCPU: 8, MemTotal: 8 * gigabyte, DockerRootDir: "/var/lib/docker",
				ServerVersion: "29.3.1", OperatingSystem: "Docker Desktop"}},
		}
		checker.CheckTools(context.Background())
		checker.CheckMachine(context.Background())

		assert.Equal(t, map[string]any{
			"os_version": "darwin 15.1", "docker_version": "29.3.1", "docker_engine": "Docker Desktop",
			"cpus": 8, "memory_gb": 8.0, "disk_free_gb": 100.0,
		}, checker.Facts())
	})

	t.Run("unreachable docker reports no fake numbers", func(t *testing.T) {
		checker := &Checker{host: fakeHost{}, docker: fakeDocker{err: errors.New("down")}}
		checker.CheckTools(context.Background())
		checker.CheckMachine(context.Background())

		assert.Equal(t, map[string]any{"os_version": "darwin 15.1"}, checker.Facts())
	})
}

func TestFreeDiskAt_OnlyMissingDirIsSkipped(t *testing.T) {
	tmp := t.TempDir()

	_, visible, err := freeDiskAt(filepath.Join(tmp, "missing"))
	assert.False(t, visible)
	assert.NoError(t, err)

	_, visible, err = freeDiskAt(tmp)
	assert.True(t, visible)
	assert.NoError(t, err)

	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions are not enforced here")
	}
	locked := filepath.Join(tmp, "locked")
	assert.NoError(t, os.MkdirAll(filepath.Join(locked, "docker"), 0o700))
	assert.NoError(t, os.Chmod(locked, 0))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	_, visible, err = freeDiskAt(filepath.Join(locked, "docker"))
	assert.True(t, visible)
	assert.Error(t, err)
}

func TestCheckTools_DockerBlocksButMissingToolsOnlyWarn(t *testing.T) {
	tests := []struct {
		name             string
		missing          map[string]bool
		dockerErr        error
		wantFailure      bool
		wantDockerDetail string
	}{
		{"docker missing", map[string]bool{"docker": true}, nil, true, "not found"},
		{"docker not running", nil, errors.New("Cannot connect to the Docker daemon"), true, "not reachable"},
		{"docker slow to answer", nil, errDockerTimeout, true, "not answering"},
		{"system socket permission denied", nil, errors.New("permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock"), true, "permission denied"},
		{"rootless socket permission denied gets no group advice", nil, errors.New("permission denied while trying to connect to the Docker daemon socket at unix:///run/user/1000/docker.sock"), true, "not reachable"},
		{"all tools missing but docker running", map[string]bool{"kubectl": true, "helm": true, "kind": true}, nil, false, "running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &Checker{
				host:   fakeHost{missing: tt.missing},
				docker: fakeDocker{err: tt.dockerErr},
			}

			results := checker.CheckTools(context.Background())

			assert.Equal(t, tt.wantFailure, HasFailure(results))
			assert.Equal(t, tt.wantDockerDetail, results[0].Detail)
		})
	}
}

func TestCheckNetwork_NamesBlockedSitesButOnlyWarns(t *testing.T) {
	checker := &Checker{host: fakeHost{blocked: map[string]bool{
		"https://registry-1.docker.io/v2/": true,
		"https://github.com/":              true,
	}}}

	r := checker.CheckNetwork(context.Background())

	assert.Equal(t, StatusWarn, r.Status)
	assert.Equal(t, "can't reach Docker Hub, github.com", r.Detail)
	assert.Equal(t, []string{"Docker Hub", "github.com"}, checker.Facts()["network_unreachable"])
}

func TestSemver_ReadsEachToolsVersionOutput(t *testing.T) {
	outputs := map[string]string{
		"Client Version: v1.37.1\nKustomize Version: v5.8.1": "v1.37.1",
		"v4.3.0+gbec5b06":                    "v4.3.0",
		"kind v0.33.0 go1.26.7 darwin/arm64": "v0.33.0",
		"command not found":                  "",
	}
	for out, want := range outputs {
		assert.Equal(t, want, semver.FindString(out), out)
	}
}

func TestCheckMachine_WarnsAtBoundaryButNeverFails(t *testing.T) {
	tests := []struct {
		name        string
		cpus        int
		memory      int64
		root        string
		free        uint64
		diskVisible bool
		readErr     error
		wantWarns   int
		wantResults int
	}{
		{"exactly the minimum", minCPUs, minMemory, "/var/lib/docker", minFreeDisk, true, nil, 0, 3},
		{"one below every minimum", minCPUs - 1, minMemoryReported - 1, "/var/lib/docker", minFreeDisk - 1, true, nil, 3, 3},
		{"6 GB setting reported as 5.8 GB passes", minCPUs, 58 * gigabyte / 10, "/var/lib/docker", minFreeDisk, true, nil, 0, 3},
		{"5.4 GB reported warns", minCPUs, 54 * gigabyte / 10, "/var/lib/docker", minFreeDisk, true, nil, 1, 3},
		{"docker info error", 0, 0, "", 0, false, errors.New("boom"), 3, 3},
		{"docker desktop disk not visible is skipped", minCPUs, minMemory, "/var/lib/docker", 0, false, nil, 0, 2},
		{"podman-style zero fields warn instead of fake numbers", 0, 0, "", 0, false, nil, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &Checker{
				host:   fakeHost{free: tt.free, diskVisible: tt.diskVisible, diskErr: tt.readErr},
				docker: fakeDocker{info: dockerInfo{NCPU: tt.cpus, MemTotal: tt.memory, DockerRootDir: tt.root}, err: tt.readErr},
			}

			results := checker.CheckMachine(context.Background())

			warns := 0
			for _, r := range results {
				if r.Status == StatusWarn {
					warns++
				}
			}
			assert.Equal(t, tt.wantWarns, warns)
			assert.Len(t, results, tt.wantResults)
			assert.False(t, HasFailure(results))
		})
	}
}
