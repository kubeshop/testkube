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
	info DockerInfo
	err  error
}

func (f fakeDocker) Info(context.Context) (DockerInfo, error) {
	return f.info, f.err
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
		{"system socket permission denied", nil, errors.New("permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock"), true, "permission denied"},
		{"rootless socket permission denied gets no group advice", nil, errors.New("permission denied while trying to connect to the Docker daemon socket at unix:///run/user/1000/docker.sock"), true, "not reachable"},
		{"all tools missing but docker running", map[string]bool{"kubectl": true, "helm": true, "kind": true}, nil, false, "/usr/bin/docker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &Checker{
				LookPath: func(name string) (string, error) {
					if tt.missing[name] {
						return "", errors.New("not found")
					}
					return "/usr/bin/" + name, nil
				},
				Docker: fakeDocker{err: tt.dockerErr},
			}

			results := checker.CheckTools(context.Background())

			assert.Equal(t, tt.wantFailure, HasFailure(results))
			assert.Equal(t, tt.wantDockerDetail, results[0].Detail)
		})
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
		{"one below every minimum", minCPUs - 1, minMemory - 1, "/var/lib/docker", minFreeDisk - 1, true, nil, 3, 3},
		{"docker info error", 0, 0, "", 0, false, errors.New("boom"), 3, 3},
		{"docker desktop disk not visible is skipped", minCPUs, minMemory, "/var/lib/docker", 0, false, nil, 0, 2},
		{"podman-style zero fields warn instead of fake numbers", 0, 0, "", 0, false, nil, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &Checker{
				Docker:     fakeDocker{info: DockerInfo{NCPU: tt.cpus, MemTotal: tt.memory, DockerRootDir: tt.root}, err: tt.readErr},
				FreeDiskAt: func(string) (uint64, bool, error) { return tt.free, tt.diskVisible, tt.readErr },
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
