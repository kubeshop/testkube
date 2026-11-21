package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// The controller manager's scheme knows every kind its controllers watch, including core/v1
// ConfigMaps, which the Skill and AIAgent sync watches by metadata: a kind missing here fails the
// controller at startup and crashloops the runner.
func TestNewControllerScheme(t *testing.T) {
	scheme, err := NewControllerScheme()
	require.NoError(t, err)
	for _, gvk := range []schema.GroupVersionKind{
		{Group: "", Version: "v1", Kind: "ConfigMap"},
		{Group: "ai.testkube.io", Version: "v1alpha1", Kind: "Skill"},
		{Group: "ai.testkube.io", Version: "v1alpha1", Kind: "AIAgent"},
		{Group: "ai.testkube.io", Version: "v1alpha1", Kind: "MCPServer"},
		{Group: "ai.testkube.io", Version: "v1alpha1", Kind: "LLMProvider"},
		{Group: "insights.testkube.io", Version: "v1alpha1", Kind: "Board"},
		{Group: "tests.testkube.io", Version: "v1", Kind: "TestTrigger"},
		{Group: "testworkflows.testkube.io", Version: "v1", Kind: "TestWorkflow"},
	} {
		assert.True(t, scheme.Recognizes(gvk), "scheme does not recognize %s", gvk)
	}
}
