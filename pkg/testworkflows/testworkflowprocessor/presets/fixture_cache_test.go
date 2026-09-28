package presets

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

// TestCacheFixturesBundle parses the shipped cache fixtures and bundles every step that
// declares a cache, so a fixture cannot be committed in a shape the processor refuses.
func TestCacheFixturesBundle(t *testing.T) {
	for _, file := range []string{
		"../../../../test/special-cases/dependency-cache.yaml",
		"../../../../test/suites/special-cases/dependency-cache-suite.yaml",
	} {
		raw, err := os.ReadFile(file)
		require.NoError(t, err, file)

		var cached int
		for _, doc := range strings.Split(string(raw), "\n---") {
			if strings.TrimSpace(doc) == "" {
				continue
			}
			var wf testworkflowsv1.TestWorkflow
			require.NoError(t, yaml.UnmarshalStrict([]byte(doc), &wf), "%s: %s", file, firstLines(doc))
			for _, step := range wf.Spec.Steps {
				if step.Cache == nil {
					continue
				}
				cached++
				res, err := bundleWithCache(t, step)
				require.NoError(t, err, "%s: %s: step %q", file, wf.Name, step.Name)
				assert.NotNil(t, res)
			}
		}
		t.Logf("%s: bundled %d cache blocks", file, cached)
		assert.Positive(t, cached, "%s: no cache blocks were found, so this checked nothing", file)
	}
}

func firstLines(doc string) string {
	lines := strings.SplitN(strings.TrimSpace(doc), "\n", 4)
	return strings.Join(lines[:min(len(lines), 3)], " | ")
}
