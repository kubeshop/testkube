package localinstall

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var movedPorts = Ports{"dashboard": 8081, "api": 8091, "login": 5557, "storage": 9001, "ai": 9091}

func TestPortValues_FollowMovedPorts(t *testing.T) {
	values := PortValues(movedPorts)

	for _, old := range []string{"localhost:8080", "localhost:8090", "localhost:5556", "localhost:9000", "localhost:9090"} {
		assert.NotContains(t, values, old)
	}
	assert.Contains(t, values, "redirectUri: http://localhost:8091/auth/callback")
	assert.Contains(t, values, "issuer: http://localhost:5557")
}

// A chart bump's new browser URL must not keep 8090.
func TestPortValues_OverrideEveryBrowserURLInTheDemoValues(t *testing.T) {
	var demo, ours map[string]any
	require.NoError(t, yaml.Unmarshal(EnterpriseDemoValues, &demo))
	require.NoError(t, yaml.Unmarshal([]byte(PortValues(movedPorts)), &ours))

	for path, value := range flatten("", demo) {
		s, ok := value.(string)
		if !ok || !strings.Contains(s, "localhost:") || strings.Contains(s, "localhost:*") {
			continue
		}
		assert.True(t, overridden(path, ours), "%s = %q is not overridden", path, s)
	}
}

func TestServicesManifest_ReachEveryBrowserPortOnItsFixedNodePort(t *testing.T) {
	manifest := ServicesManifest()

	for _, cp := range clusterPorts {
		assert.Contains(t, manifest, fmt.Sprintf("nodePort: %d\n", cp.inNodes), cp.name)
	}
	assert.Equal(t, len(clusterPorts), strings.Count(manifest, "app.kubernetes.io/instance: "+ReleaseName))
}

func flatten(prefix string, v any) map[string]any {
	out := map[string]any{}
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			for p, leaf := range flatten(prefix+"."+k, child) {
				out[p] = leaf
			}
		}
	case []any:
		for i, child := range t {
			for p, leaf := range flatten(fmt.Sprintf("%s[%d]", prefix, i), child) {
				out[p] = leaf
			}
		}
	default:
		out[prefix] = v
	}
	return out
}

// Helm replaces lists whole; setting one covers its items.
func overridden(path string, ours map[string]any) bool {
	for p := range flatten("", ours) {
		if p == path {
			return true
		}
		if root, _, isList := strings.Cut(p, "["); isList && strings.HasPrefix(path, root+"[") {
			return true
		}
	}
	return false
}
