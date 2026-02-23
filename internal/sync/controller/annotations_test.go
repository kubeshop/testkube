package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasNoGitOpsSyncAnnotation(t *testing.T) {
	testCases := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{
			name:        "missing annotation",
			annotations: nil,
			expected:    false,
		},
		{
			name: "annotation true",
			annotations: map[string]string{
				noGitOpsSyncAnnotation: "true",
			},
			expected: true,
		},
		{
			name: "annotation false",
			annotations: map[string]string{
				noGitOpsSyncAnnotation: "false",
			},
			expected: false,
		},
		{
			name: "annotation empty",
			annotations: map[string]string{
				noGitOpsSyncAnnotation: "",
			},
			expected: false,
		},
		{
			name: "annotation invalid",
			annotations: map[string]string{
				noGitOpsSyncAnnotation: "definitely-not-bool",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj := &metav1.ObjectMeta{Annotations: tc.annotations}
			if actual := hasNoGitOpsSyncAnnotation(obj); actual != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, actual)
			}
		})
	}
}
