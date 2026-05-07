package informer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestPathMatches(t *testing.T) {
	tests := []struct {
		paths    []string
		file     string
		expected bool
	}{
		{[]string{"src"}, "src/main.go", true},
		{[]string{"src"}, "src", true},
		{[]string{"src/"}, "src/main.go", true},
		{[]string{"other"}, "src/main.go", false},
		{[]string{"src", "pkg"}, "pkg/util.go", true},
		{[]string{""}, "anything.go", false},
		{[]string{"src/sub"}, "src/sub/file.go", true},
		{[]string{"src/sub"}, "src/other/file.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			assert.Equal(t, tt.expected, pathMatches(tt.paths, tt.file))
		})
	}
}

func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "refs/heads/main"},
		{"refs/heads/main", "refs/heads/main"},
		{"refs/tags/v1.0", "refs/tags/v1.0"},
		{"", ""},
		{"  develop  ", "refs/heads/develop"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeRef(tt.input))
		})
	}
}

func TestIsGitContentTrigger(t *testing.T) {
	resource := testkube.CONTENT_TestTriggerResources

	tests := []struct {
		name     string
		trigger  testkube.TestTrigger
		expected bool
	}{
		{
			name: "valid git content trigger",
			trigger: testkube.TestTrigger{
				Resource: &resource,
				ContentSelector: &testkube.TestTriggerContentSelector{
					Git: &testkube.TestTriggerContentGit{
						Uri: "https://github.com/example/repo.git",
					},
				},
			},
			expected: true,
		},
		{
			name: "disabled trigger",
			trigger: testkube.TestTrigger{
				Disabled: true,
				Resource: &resource,
				ContentSelector: &testkube.TestTriggerContentSelector{
					Git: &testkube.TestTriggerContentGit{
						Uri: "https://github.com/example/repo.git",
					},
				},
			},
			expected: false,
		},
		{
			name:     "no resource",
			trigger:  testkube.TestTrigger{},
			expected: false,
		},
		{
			name: "no content selector",
			trigger: testkube.TestTrigger{
				Resource: &resource,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isGitContentTrigger(tt.trigger))
		})
	}
}
