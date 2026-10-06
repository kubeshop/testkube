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
			restoreLookPath, restoreDocker := lookPath, dockerReachable
			t.Cleanup(func() { lookPath, dockerReachable = restoreLookPath, restoreDocker })
			lookPath = func(name string) (string, error) {
				if tt.missing[name] {
					return "", errors.New("not found")
				}
				return "/usr/bin/" + name, nil
			}
			dockerReachable = func(context.Context) error { return tt.dockerErr }

			results := CheckTools(context.Background())

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
		free        uint64
		diskVisible bool
		readErr     error
		wantWarns   int
		wantResults int
	}{
		{"exactly the minimum", minCPUs, minMemory, minFreeDisk, true, nil, 0, 3},
		{"one below every minimum", minCPUs - 1, minMemory - 1, minFreeDisk - 1, true, nil, 3, 3},
		{"docker and disk unreadable", 0, 0, 0, false, errors.New("boom"), 3, 3},
		{"docker desktop disk not visible is skipped", minCPUs, minMemory, 0, false, nil, 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreResources, restoreDisk := dockerResources, dockerFreeDiskBytes
			t.Cleanup(func() { dockerResources, dockerFreeDiskBytes = restoreResources, restoreDisk })
			dockerResources = func(context.Context) (int, int64, error) { return tt.cpus, tt.memory, tt.readErr }
			dockerFreeDiskBytes = func(context.Context) (uint64, bool, error) { return tt.free, tt.diskVisible, tt.readErr }

			results := CheckMachine(context.Background())

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
