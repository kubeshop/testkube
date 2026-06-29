package triggers

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubeshop/testkube/pkg/utils"
)

func TestNewUnstructuredTemplateObject(t *testing.T) {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "my-rollout",
				"namespace": "production",
				"labels": map[string]interface{}{
					"app":                        "myapp",
					"tags.datadoghq.com/version": "1.2.3",
				},
				"annotations": map[string]interface{}{
					"some-annotation": "some-value",
				},
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"strategy": map[string]interface{}{
					"canary": map[string]interface{}{
						"steps": []interface{}{},
					},
				},
			},
			"status": map[string]interface{}{
				"phase": "Healthy",
			},
		},
	}

	obj := newUnstructuredTemplateObject(u)

	// Verify ObjectMeta fields
	objectMeta, ok := (*obj)["ObjectMeta"].(metav1.ObjectMeta)
	require.True(t, ok)
	assert.Equal(t, "my-rollout", objectMeta.Name)
	assert.Equal(t, "production", objectMeta.Namespace)
	assert.Equal(t, map[string]string{
		"app":                        "myapp",
		"tags.datadoghq.com/version": "1.2.3",
	}, objectMeta.Labels)
	assert.Equal(t, map[string]string{
		"some-annotation": "some-value",
	}, objectMeta.Annotations)

	// Verify Spec and Status
	spec, ok := (*obj)["Spec"].(map[string]interface{})
	require.True(t, ok)
	status, ok := (*obj)["Status"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, int64(3), spec["replicas"])
	assert.Equal(t, "Healthy", status["phase"])

	// Verify Kind and APIVersion
	assert.Equal(t, "Rollout", (*obj)["Kind"])
	assert.Equal(t, "argoproj.io/v1alpha1", (*obj)["APIVersion"])

	// Verify legacy raw map access compatibility
	metadata, ok := (*obj)["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-rollout", metadata["name"])
}

func TestUnstructuredTemplateObject_GoTemplate(t *testing.T) {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      "my-rollout",
				"namespace": "production",
				"labels": map[string]interface{}{
					"app":                        "myapp",
					"tags.datadoghq.com/version": "1.2.3",
				},
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
			},
		},
	}

	obj := newUnstructuredTemplateObject(u)

	tests := []struct {
		name     string
		tmpl     string
		expected string
	}{
		{
			name:     "access ObjectMeta.Labels with index",
			tmpl:     `{{ index .ObjectMeta.Labels "tags.datadoghq.com/version" }}`,
			expected: "1.2.3",
		},
		{
			name:     "access ObjectMeta.Labels simple key",
			tmpl:     `{{ index .ObjectMeta.Labels "app" }}`,
			expected: "myapp",
		},
		{
			name:     "access ObjectMeta.Name",
			tmpl:     `{{ .ObjectMeta.Name }}`,
			expected: "my-rollout",
		},
		{
			name:     "access ObjectMeta.Namespace",
			tmpl:     `{{ .ObjectMeta.Namespace }}`,
			expected: "production",
		},
		{
			name:     "access Kind",
			tmpl:     `{{ .Kind }}`,
			expected: "Rollout",
		},
		{
			name:     "access legacy metadata.name",
			tmpl:     `{{ .metadata.name }}`,
			expected: "my-rollout",
		},
		{
			name:     "access legacy metadata.labels with index",
			tmpl:     `{{ index .metadata.labels "app" }}`,
			expected: "myapp",
		},
		{
			name:     "access TypeMeta.Kind",
			tmpl:     `{{ .TypeMeta.Kind }}`,
			expected: "Rollout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := utils.NewTemplate("test").Parse(tt.tmpl)
			require.NoError(t, err)

			var buf bytes.Buffer
			err = tmpl.ExecuteTemplate(&buf, "test", obj)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestUnstructuredTemplateObject_NilMetadata(t *testing.T) {
	// Test with minimal unstructured object (no labels, no annotations)
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "example.com/v1",
			"kind":       "Custom",
			"metadata": map[string]interface{}{
				"name":      "minimal",
				"namespace": "default",
			},
		},
	}

	obj := newUnstructuredTemplateObject(u)
	objectMeta, ok := (*obj)["ObjectMeta"].(metav1.ObjectMeta)
	require.True(t, ok)
	assert.Equal(t, "minimal", objectMeta.Name)
	assert.Nil(t, objectMeta.Labels)
	assert.Nil(t, objectMeta.Annotations)
	assert.Nil(t, (*obj)["Spec"])
	assert.Nil(t, (*obj)["Status"])
}

func TestUnstructuredTemplateObject_MatchesDeploymentBehavior(t *testing.T) {
	// This test verifies that the same template works for both a typed
	// Deployment (passed as object) and a CRD via unstructuredTemplateObject
	tmplStr := `{{ index .ObjectMeta.Labels "tags.datadoghq.com/version" }}`

	// Simulate what happens for a Deployment
	type fakeDeployment struct {
		ObjectMeta metav1.ObjectMeta
	}
	deployment := &fakeDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"tags.datadoghq.com/version": "2.0.0",
			},
		},
	}

	tmpl, err := utils.NewTemplate("test").Parse(tmplStr)
	require.NoError(t, err)

	var deployBuf bytes.Buffer
	err = tmpl.ExecuteTemplate(&deployBuf, "test", deployment)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", deployBuf.String())

	// Now simulate what happens for a CRD via unstructuredTemplateObject
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name": "my-rollout",
				"labels": map[string]interface{}{
					"tags.datadoghq.com/version": "2.0.0",
				},
			},
		},
	}
	crdObj := newUnstructuredTemplateObject(u)

	tmpl2, err := utils.NewTemplate("test").Parse(tmplStr)
	require.NoError(t, err)

	var crdBuf bytes.Buffer
	err = tmpl2.ExecuteTemplate(&crdBuf, "test", crdObj)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", crdBuf.String())

	// Both should produce the same result
	assert.Equal(t, deployBuf.String(), crdBuf.String())
}
