package pro

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelmRelease_JSONParsing(t *testing.T) {
	input := `[
		{"name":"testkube","namespace":"testkube","chart":"testkube-2.1.0"},
		{"name":"my-runner","namespace":"runners","chart":"testkube-runner-1.0.0"},
		{"name":"other","namespace":"default","chart":"nginx-1.0.0"}
	]`

	var releases []helmRelease
	err := json.Unmarshal([]byte(input), &releases)
	require.NoError(t, err)
	assert.Len(t, releases, 3)

	assert.Equal(t, "testkube", releases[0].Name)
	assert.Equal(t, "testkube", releases[0].Namespace)
	assert.Equal(t, "testkube-2.1.0", releases[0].Chart)

	assert.Equal(t, "my-runner", releases[1].Name)
	assert.Equal(t, "runners", releases[1].Namespace)
	assert.Equal(t, "testkube-runner-1.0.0", releases[1].Chart)
}

func TestHelmRelease_FindRunnerChartPrefix(t *testing.T) {
	tests := []struct {
		name        string
		releases    []helmRelease
		expectName  string
		expectNs    string
		expectFound bool
	}{
		{
			name: "runner release found",
			releases: []helmRelease{
				{Name: "testkube", Namespace: "testkube", Chart: "testkube-2.1.0"},
				{Name: "default-oss", Namespace: "testkube", Chart: "testkube-runner-1.0.0"},
			},
			expectName:  "default-oss",
			expectNs:    "testkube",
			expectFound: true,
		},
		{
			name: "no runner release",
			releases: []helmRelease{
				{Name: "testkube", Namespace: "testkube", Chart: "testkube-2.1.0"},
				{Name: "nginx", Namespace: "default", Chart: "nginx-1.0.0"},
			},
			expectName:  "",
			expectNs:    "",
			expectFound: false,
		},
		{
			name:        "empty list",
			releases:    []helmRelease{},
			expectName:  "",
			expectNs:    "",
			expectFound: false,
		},
		{
			name: "multiple runner releases returns first",
			releases: []helmRelease{
				{Name: "runner-1", Namespace: "ns-1", Chart: "testkube-runner-1.0.0"},
				{Name: "runner-2", Namespace: "ns-2", Chart: "testkube-runner-2.0.0"},
			},
			expectName:  "runner-1",
			expectNs:    "ns-1",
			expectFound: true,
		},
		{
			name: "chart name similar but not matching prefix",
			releases: []helmRelease{
				{Name: "fake", Namespace: "default", Chart: "my-testkube-runner-1.0.0"},
			},
			expectName:  "",
			expectNs:    "",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var foundName, foundNs string
			found := false
			for _, r := range tt.releases {
				if strings.HasPrefix(r.Chart, "testkube-runner-") {
					foundName = r.Name
					foundNs = r.Namespace
					found = true
					break
				}
			}
			assert.Equal(t, tt.expectFound, found)
			assert.Equal(t, tt.expectName, foundName)
			assert.Equal(t, tt.expectNs, foundNs)
		})
	}
}

func TestContextDescription(t *testing.T) {
	assert.Equal(t, "Open Source Testkube", contextDescription["oss"])
	assert.Equal(t, "Testkube in Pro mode", contextDescription["cloud"])
	assert.Contains(t, contextDescription[""], "Unknown context")
}
