package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveImageReference(t *testing.T) {
	t.Parallel()

	opts := Options{
		HelmSet: map[string]string{
			"image.registry":   "registry.depot.dev",
			"image.repository": "8m4mpphf9g",
			"image.tag":        "ecd4d87",
			"image.tagSuffix":  "-usage-export",
		},
	}
	got := resolveImageReference(opts, "2.12.0-rc.0")
	want := "registry.depot.dev/8m4mpphf9g:ecd4d87-usage-export"
	if got != want {
		t.Fatalf("resolveImageReference() = %q, want %q", got, want)
	}
}

func TestResolveChartVersionFromPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(`version: 1.2.3
appVersion: 9.9.9
`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := resolveChartVersion(Options{ChartPath: dir})
	if !strings.HasPrefix(got, "1.2.3 (") {
		t.Fatalf("resolveChartVersion() = %q", got)
	}
}

func TestHelmSetValue(t *testing.T) {
	t.Parallel()

	set := map[string]string{"image.tag": "dev"}
	if got := helmSetValue(set, "image.tag", "fallback"); got != "dev" {
		t.Fatalf("helmSetValue() = %q", got)
	}
	if got := helmSetValue(set, "image.registry", "fallback"); got != "fallback" {
		t.Fatalf("helmSetValue() = %q", got)
	}
}
