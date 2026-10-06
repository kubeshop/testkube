package localinstall

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckTools_DockerBlocksButMissingToolsOnlyWarn(t *testing.T) {
	tests := []struct {
		name          string
		missing       map[string]bool
		dockerRunning bool
		wantFailure   bool
	}{
		{"docker missing", map[string]bool{"docker": true}, false, true},
		{"docker not running", nil, false, true},
		{"all tools missing but docker running", map[string]bool{"kubectl": true, "helm": true, "kind": true}, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreLookPath, restoreDocker := lookPath, dockerRunning
			t.Cleanup(func() { lookPath, dockerRunning = restoreLookPath, restoreDocker })
			lookPath = func(name string) (string, error) {
				if tt.missing[name] {
					return "", errors.New("not found")
				}
				return "/usr/bin/" + name, nil
			}
			dockerRunning = func(context.Context) bool { return tt.dockerRunning }

			assert.Equal(t, tt.wantFailure, HasFailure(CheckTools(context.Background())))
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
