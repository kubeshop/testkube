package common

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

func TestProcessMasterFlags_UiUri(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		cfg        *config.Data
		expectedUi string
	}{
		{
			name:       "no flags keep the default UI URI",
			args:       nil,
			expectedUi: "https://app.testkube.io",
		},
		{
			name:       "api uri override alone clears the UI URI",
			args:       []string{"--api-uri-override", "https://api.onprem.example.com"},
			expectedUi: "",
		},
		{
			name: "api uri override with ui uri override keeps the UI URI",
			args: []string{
				"--api-uri-override", "https://api.onprem.example.com",
				"--ui-uri-override", "https://dashboard.onprem.example.com",
			},
			expectedUi: "https://dashboard.onprem.example.com",
		},
		{
			name: "api uri override with root domain keeps the composed UI URI",
			args: []string{
				"--api-uri-override", "https://api.onprem.example.com",
				"--root-domain", "onprem.example.com",
			},
			expectedUi: "https://app.onprem.example.com",
		},
		{
			name: "api uri override with ui prefix keeps the composed UI URI",
			args: []string{
				"--api-uri-override", "https://api.onprem.example.com",
				"--ui-prefix", "dashboard",
			},
			expectedUi: "https://dashboard.testkube.io",
		},
		{
			name: "api uri override with a saved root domain keeps the composed UI URI",
			args: []string{"--api-uri-override", "https://api.onprem.example.com"},
			cfg: &config.Data{
				Master: config.Master{RootDomain: "onprem.example.com"},
			},
			expectedUi: "https://app.onprem.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts HelmOptions
			cmd := &cobra.Command{Use: "test"}
			PopulateMasterFlags(cmd, &opts, false)
			require.NoError(t, cmd.Flags().Parse(tt.args))

			ProcessMasterFlags(cmd, &opts, tt.cfg)

			assert.Equal(t, tt.expectedUi, opts.Master.URIs.Ui)
		})
	}
}

func TestControlPlaneAPIURI(t *testing.T) {
	const (
		contextURI = "https://api.onprem.example.com"
		saasURI    = "https://api.testkube.io"
	)

	onPrem := &config.Data{
		CloudContext: config.CloudContext{ApiUri: contextURI},
	}

	tests := []struct {
		name     string
		args     []string
		cfg      *config.Data
		expected string
	}{
		{
			name:     "the saved control plane wins when nothing is configured",
			cfg:      onPrem,
			expected: contextURI,
		},
		{
			name:     "an explicit api uri override wins over the saved one",
			args:     []string{"--api-uri-override", "https://api.other.example.com"},
			cfg:      onPrem,
			expected: "https://api.other.example.com",
		},
		{
			name:     "an explicit root domain wins over the saved one",
			args:     []string{"--root-domain", "other.example.com"},
			cfg:      onPrem,
			expected: "https://api.other.example.com",
		},
		{
			name:     "a deprecated root domain flag counts as explicit",
			args:     []string{"--cloud-root-domain", "other.example.com"},
			cfg:      onPrem,
			expected: "https://api.other.example.com",
		},
		{
			name:     "an api prefix counts as explicit",
			args:     []string{"--api-prefix", "gateway"},
			cfg:      onPrem,
			expected: "https://gateway.testkube.io",
		},
		{
			name:     "the composed default is used with no saved control plane",
			cfg:      &config.Data{},
			expected: saasURI,
		},
		{
			name:     "a nil config falls back to the composed default",
			cfg:      nil,
			expected: saasURI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts HelmOptions
			cmd := &cobra.Command{Use: "test"}
			PopulateMasterFlags(cmd, &opts, false)
			require.NoError(t, cmd.Flags().Parse(tt.args))

			ProcessMasterFlags(cmd, &opts, tt.cfg)

			assert.Equal(t, tt.expected, ControlPlaneAPIURI(cmd, opts.Master.URIs.Api, tt.cfg))
		})
	}
}
