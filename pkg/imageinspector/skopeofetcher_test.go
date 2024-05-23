package imageinspector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractRegistry uses table-driven tests to validate the extractRegistry function.
func TestExtractRegistry(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{"DockerHub short", "nginx:latest", "https://index.docker.io/v1/"},
		{"DockerHub long", "library/nginx:latest", "https://index.docker.io/v1/"},
		{"GCR", "gcr.io/google-containers/busybox:latest", "gcr.io"},
		{"ECR", "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-application:latest", "123456789012.dkr.ecr.us-east-1.amazonaws.com"},
		{"MCR", "mcr.microsoft.com/dotnet/core/sdk:3.1", "mcr.microsoft.com"},
		{"Quay", "quay.io/bitnami/nginx:latest", "quay.io"},
		{"Custom port", "localhost:5000/myimage:latest", "localhost:5000"},
		{"No tag", "myregistry.com/myimage", "myregistry.com"},
		{"Only image", "myimage", "https://index.docker.io/v1/"},
		{"Custom GitLab", "registry.gitlab.com/company/base-docker-images/ubuntu-python-base-image:3.12.0-jammy", "registry.gitlab.com"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := extractRegistry(tc.image)
			assert.Equal(t, tc.expected, got)
		})
	}
}
