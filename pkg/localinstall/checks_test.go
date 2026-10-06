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
