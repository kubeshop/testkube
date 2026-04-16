package jsonpath

import "testing"

func TestIsRootLevelFilter(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "root filter with equality",
			path:     "$[?(@.spec.execution.silent==true)]",
			expected: true,
		},
		{
			name:     "root filter with trailing path",
			path:     "$[?(@.spec.execution.silent==true)].metadata.name",
			expected: true,
		},
		{
			name:     "root filter with string comparison",
			path:     "$[?(@.metadata.name=='my-workflow')]",
			expected: true,
		},
		{
			name:     "root filter with inequality",
			path:     "$[?(@.spec.steps.length > 3)]",
			expected: true,
		},
		{
			name:     "simple property path",
			path:     "$.spec.execution.silent",
			expected: false,
		},
		{
			name:     "nested filter on array field",
			path:     "$.steps[?(@.status=='failed')]",
			expected: false,
		},
		{
			name:     "nested filter with wildcard prefix",
			path:     "$[*].steps[?(@.status=='failed')]",
			expected: false,
		},
		{
			name:     "nested filter with index prefix",
			path:     "$[0].steps[?(@.name=='test')]",
			expected: false,
		},
		{
			name:     "recursive descent",
			path:     "$..image",
			expected: false,
		},
		{
			name:     "wildcard at root",
			path:     "$[*]",
			expected: false,
		},
		{
			name:     "invalid expression",
			path:     "$[?(@.invalid[[[",
			expected: false,
		},
		{
			name:     "empty expression",
			path:     "",
			expected: false,
		},
		{
			name:     "root only",
			path:     "$",
			expected: false,
		},
		{
			name:     "chained root-level filters",
			path:     "$[?(@.x)][?(@.y)]",
			expected: true,
		},
		{
			name:     "at-sign root is not dollar root",
			path:     "@[?(@.x=='y')]",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRootLevelFilter(tt.path)
			if result != tt.expected {
				t.Errorf("IsRootLevelFilter(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
