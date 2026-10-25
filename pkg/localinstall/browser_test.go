package localinstall

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrowserCommand_LinuxPicksWhatCanReallyOpenIt(t *testing.T) {
	tests := map[string]struct {
		kernel  string
		env     map[string]string
		wslview bool
		want    []string
	}{
		"desktop":                 {"6.8.0-45-generic", map[string]string{"DISPLAY": ":0"}, false, []string{"xdg-open", "u"}},
		"wayland desktop":         {"6.8.0-45-generic", map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, false, []string{"xdg-open", "u"}},
		"server without a screen": {"6.8.0-45-generic", nil, false, nil},
		"WSL with wslview":        {"5.15.167.4-microsoft-standard-WSL2", map[string]string{"DISPLAY": ":0"}, true, []string{"wslview", "u"}},
		"WSL without wslview":     {"5.15.167.4-microsoft-standard-WSL2", nil, false, []string{"explorer.exe", "u"}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			lookPath := func(string) (string, error) {
				if tt.wslview {
					return "/usr/bin/wslview", nil
				}
				return "", errors.New("not found")
			}

			got := browserCommand("linux", func(k string) string { return tt.env[k] }, tt.kernel, lookPath, "u")

			assert.Equal(t, tt.want, got)
		})
	}
}
