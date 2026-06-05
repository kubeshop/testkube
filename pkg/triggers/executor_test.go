package triggers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestGetTemplateData_MetadataLabels verifies that Go templates in v1
// TestTrigger actionParameters work with BOTH JSON-style field names
// (.metadata.labels) and Go struct field names (.ObjectMeta.Labels),
// regardless of whether the watcher delivers a typed Go struct (built-in
// resources like deployment) or an unstructured map (custom resources
// via resourceRef).
func TestGetTemplateData_MetadataLabels(t *testing.T) {
	s := &Service{}

	// Simulate a built-in resource (resource: deployment)
	// The watcher passes a typed Go struct
	typedDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "default",
			Labels: map[string]string{
				"tags.datadoghq.com/version": "1.0.0",
			},
		},
	}

	// Simulate a custom resource (resourceRef:)
	// The dynamic informer passes an unstructured map
	unstructuredObject := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "my-app",
			"namespace": "default",
			"labels": map[string]interface{}{
				"tags.datadoghq.com/version": "1.0.0",
			},
		},
	}

	// Template using JSON-style field names (what users naturally write)
	jsonTemplate := `{{ index .metadata.labels "tags.datadoghq.com/version" }}`

	// Template using Go struct field names (legacy workaround for built-in resources)
	goTemplate := `{{ index .ObjectMeta.Labels "tags.datadoghq.com/version" }}`

	// TEST 1: JSON-style template with unstructured map
	t.Run("json_template_with_unstructured_map", func(t *testing.T) {
		e := &watcherEvent{Object: unstructuredObject}
		result, err := s.getTemplateData(e, jsonTemplate)
		assert.NoError(t, err)
		assert.Equal(t, "1.0.0", string(result))
	})

	// TEST 2: JSON-style template with typed struct
	t.Run("json_template_with_typed_struct", func(t *testing.T) {
		e := &watcherEvent{Object: typedDeployment}
		result, err := s.getTemplateData(e, jsonTemplate)
		assert.NoError(t, err, ".metadata.labels should now work with typed structs via JSON normalization fallback")
		assert.Equal(t, "1.0.0", string(result))
	})

	// TEST 3: Go struct template with typed struct — still WORKS (backward compatible)
	t.Run("go_template_with_typed_struct", func(t *testing.T) {
		e := &watcherEvent{Object: typedDeployment}
		result, err := s.getTemplateData(e, goTemplate)
		assert.NoError(t, err)
		assert.Equal(t, "1.0.0", string(result))
	})

	// TEST 4: Go struct template with unstructured map — still FAILS (expected)
	t.Run("go_template_with_unstructured_map_fails", func(t *testing.T) {
		e := &watcherEvent{Object: unstructuredObject}
		_, err := s.getTemplateData(e, goTemplate)
		// .ObjectMeta doesn't exist on a map — this is expected and unchanged
		assert.Error(t, err, "Go struct field names don't work with unstructured maps")
	})
}

// TestGetTemplateData_SimpleFields verifies basic template resolution for
// common patterns users would write.
func TestGetTemplateData_SimpleFields(t *testing.T) {
	s := &Service{}

	typedDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "production",
			Labels: map[string]string{
				"app":     "my-app",
				"version": "2.0.0",
			},
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "json_style_name",
			template: `{{ .metadata.name }}`,
			expected: "my-app",
		},
		{
			name:     "json_style_namespace",
			template: `{{ .metadata.namespace }}`,
			expected: "production",
		},
		{
			name:     "json_style_simple_label",
			template: `{{ index .metadata.labels "app" }}`,
			expected: "my-app",
		},
		{
			name:     "go_style_name",
			template: `{{ .ObjectMeta.Name }}`,
			expected: "my-app",
		},
		{
			name:     "go_style_namespace",
			template: `{{ .ObjectMeta.Namespace }}`,
			expected: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &watcherEvent{Object: typedDeployment}
			result, err := s.getTemplateData(e, tt.template)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}
