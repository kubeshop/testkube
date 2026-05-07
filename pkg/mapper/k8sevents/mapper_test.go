package k8sevents

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil labels",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty labels",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "valid labels",
			input: map[string]string{
				"app":                   "testkube",
				"environment":           "production",
				"testkube.io/test-name": "my-test",
			},
			expected: map[string]string{
				"app":                   "testkube",
				"environment":           "production",
				"testkube.io/test-name": "my-test",
			},
		},
		{
			name: "labels with fullwidth periods (issue from bug report)",
			input: map[string]string{
				"acme．com/incident-policy":    "critical",
				"acme．com/owner":              "team-platform",
				"argocd．argoproj．io/instance": "my-app",
			},
			expected: map[string]string{
				"acme-com/incident-policy":    "critical",
				"acme-com/owner":              "team-platform",
				"argocd-argoproj-io/instance": "my-app",
			},
		},
		{
			name: "label value with invalid characters",
			input: map[string]string{
				"app": "test@app#name",
			},
			expected: map[string]string{
				"app": "test-app-name",
			},
		},
		{
			name: "label key with invalid characters in prefix",
			input: map[string]string{
				"test@domain.com/app": "value",
			},
			expected: map[string]string{
				"test-domain.com/app": "value",
			},
		},
		{
			name: "label with value starting and ending with hyphens",
			input: map[string]string{
				"app": "-test-app-",
			},
			expected: map[string]string{
				"app": "test-app",
			},
		},
		{
			name: "label with empty value after sanitization",
			input: map[string]string{
				"app": "---",
			},
			expected: map[string]string{},
		},
		{
			name: "label with invalid key that becomes empty after sanitization",
			input: map[string]string{
				"@@@": "value",
			},
			expected: map[string]string{},
		},
		{
			name: "label with very long value",
			input: map[string]string{
				"app": strings.Repeat("a", 100),
			},
			expected: map[string]string{
				"app": strings.Repeat("a", 63), // LabelValueMaxLength is 63
			},
		},
		{
			name: "mixed valid and invalid labels",
			input: map[string]string{
				"valid-key":                 "valid-value",
				"acme．com/invalid-key":      "value",
				"another-key":               "@@@",
				"testkube.io/workflow-name": "my-workflow",
			},
			expected: map[string]string{
				"valid-key":                 "valid-value",
				"acme-com/invalid-key":      "value",
				"testkube.io/workflow-name": "my-workflow",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLabels(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeLabelKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty key",
			input:    "",
			expected: "",
		},
		{
			name:     "valid simple key",
			input:    "app",
			expected: "app",
		},
		{
			name:     "valid qualified key",
			input:    "kubernetes.io/app-name",
			expected: "kubernetes.io/app-name",
		},
		{
			name:     "key with fullwidth period in prefix",
			input:    "acme．com/app",
			expected: "acme-com/app",
		},
		{
			name:     "key with multiple fullwidth periods",
			input:    "argocd．argoproj．io/instance",
			expected: "argocd-argoproj-io/instance",
		},
		{
			name:     "key with invalid characters in prefix",
			input:    "test@domain.com/name",
			expected: "test-domain.com/name",
		},
		{
			name:     "key with invalid characters in name part",
			input:    "domain.com/test@name",
			expected: "domain.com/test-name",
		},
		{
			name:     "key with only invalid characters",
			input:    "@@@/###",
			expected: "",
		},
		{
			name:     "key without prefix but with slash",
			input:    "/name",
			expected: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLabelKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeLabelValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty value",
			input:    "",
			expected: "",
		},
		{
			name:     "valid value",
			input:    "my-app-123",
			expected: "my-app-123",
		},
		{
			name:     "value with invalid characters",
			input:    "test@app#name",
			expected: "test-app-name",
		},
		{
			name:     "value with leading and trailing hyphens",
			input:    "-test-",
			expected: "test",
		},
		{
			name:     "value with only invalid characters",
			input:    "@@@",
			expected: "",
		},
		{
			name:     "value with underscores and dots",
			input:    "test_app.name",
			expected: "test_app.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLabelValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeDNSSubdomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "valid subdomain",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "subdomain with fullwidth periods",
			input:    "acme．com",
			expected: "acme-com",
		},
		{
			name:     "subdomain with multiple fullwidth periods",
			input:    "argocd．argoproj．io",
			expected: "argocd-argoproj-io",
		},
		{
			name:     "subdomain with invalid characters",
			input:    "test@domain.com",
			expected: "test-domain.com",
		},
		{
			name:     "subdomain with leading/trailing hyphens",
			input:    "-example.com-",
			expected: "example.com",
		},
		{
			name:     "only invalid characters",
			input:    "@@@",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeDNSSubdomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeDNSLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "valid label",
			input:    "my-app",
			expected: "my-app",
		},
		{
			name:     "label with invalid characters",
			input:    "test@app",
			expected: "test-app",
		},
		{
			name:     "label with leading/trailing hyphens",
			input:    "-test-",
			expected: "test",
		},
		{
			name:     "only invalid characters",
			input:    "@@@",
			expected: "",
		},
		{
			name:     "label with dots (invalid for DNS label)",
			input:    "test.app",
			expected: "test-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeDNSLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
