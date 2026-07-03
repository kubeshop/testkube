package imageinspector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestExtractRegistry uses table-driven tests to validate the extractRegistry function.
func TestExtractRegistry(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{"DockerHub short", "nginx:latest", ""},
		{"DockerHub long", "library/nginx:latest", ""},
		{"GCR", "gcr.io/google-containers/busybox:latest", "gcr.io"},
		{"ECR", "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-application:latest", "123456789012.dkr.ecr.us-east-1.amazonaws.com"},
		{"MCR", "mcr.microsoft.com/dotnet/core/sdk:3.1", "mcr.microsoft.com"},
		{"Quay", "quay.io/bitnami/nginx:latest", "quay.io"},
		{"Custom port", "localhost:5000/myimage:latest", "localhost:5000"},
		{"No tag", "myregistry.com/myimage", "myregistry.com"},
		{"Only image", "myimage", ""},
		{"Custom GitLab", "registry.gitlab.com/company/base-docker-images/ubuntu-python-base-image:3.12.0-jammy", "registry.gitlab.com"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractRegistry(tc.image)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestParseSecretData(t *testing.T) {

	t.Run("parse docker config base auth", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://index.docker.io/v1/\": {\"auth\": \"ZG9ja2VyLXVzZXJuYW1lOnlvdXItcmVhbGx5LXJlYWxseS1sb25nLWF1dGgta2V5\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "https://index.docker.io/v1/", "image")

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "docker-username", out[0].Username)
		assert.Equal(t, "your-really-really-long-auth-key", out[0].Password)
		assert.NoError(t, err)
	})

	t.Run("parse docker config map", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockercfg": []byte("{\"https://index.docker.io/v1/\": {\"auth\": \"ZG9ja2VyLXVzZXJuYW1lOnlvdXItcmVhbGx5LXJlYWxseS1sb25nLWF1dGgta2V5\"}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "https://index.docker.io/v1/", "image")

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "docker-username", out[0].Username)
		assert.Equal(t, "your-really-really-long-auth-key", out[0].Password)
		assert.NoError(t, err)
	})

	t.Run("parse docker config plain credentials", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://index.docker.io/v1/\": {\"username\": \"plainuser\", \"password\": \"plainpass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "https://index.docker.io/v1/", "image")

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "plainuser", out[0].Username)
		assert.Equal(t, "plainpass", out[0].Password)
		assert.NoError(t, err)
	})

	t.Run("parse docker config path credentials", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"registry.gitlab.com/company\": {\"username\": \"plainuser\", \"password\": \"plainpass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "registry.gitlab.com", "registry.gitlab.com/company/image")

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "plainuser", out[0].Username)
		assert.Equal(t, "plainpass", out[0].Password)
		assert.NoError(t, err)
	})

	t.Run("parse docker config longest path credentials", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"registry.gitlab.com/company/path\": {\"username\": \"plainuser\", \"password\": \"plainpass\"}, \"registry.gitlab.com/company\": {\"username\": \"user\", \"password\": \"pass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "registry.gitlab.com", "registry.gitlab.com/company/path/image")

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "plainuser", out[0].Username)
		assert.Equal(t, "plainpass", out[0].Password)
		assert.NoError(t, err)
	})

	t.Run("parse docker config scheme-prefixed auth key against bare registry", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://artifactory.example.com\": {\"username\": \"plainuser\", \"password\": \"plainpass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "artifactory.example.com", "artifactory.example.com/docker/library/node:latest")

		if assert.NoError(t, err) && assert.Len(t, out, 1) {
			assert.Equal(t, "plainuser", out[0].Username)
			assert.Equal(t, "plainpass", out[0].Password)
		}
	})

	t.Run("path-scoped credential wins over exact registry credential", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"registry.gitlab.com\": {\"username\": \"registryuser\", \"password\": \"registrypass\"}, \"registry.gitlab.com/company/path\": {\"username\": \"pathuser\", \"password\": \"pathpass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "registry.gitlab.com", "registry.gitlab.com/company/path/image")

		assert.NoError(t, err)
		assert.Equal(t, 1, len(out))
		assert.Equal(t, "pathuser", out[0].Username)
		assert.Equal(t, "pathpass", out[0].Password)
	})

	t.Run("exact registry credential used when no path-scoped key matches the image", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"registry.gitlab.com\": {\"username\": \"registryuser\", \"password\": \"registrypass\"}, \"registry.gitlab.com/company/path\": {\"username\": \"pathuser\", \"password\": \"pathpass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "registry.gitlab.com", "registry.gitlab.com/other/image")

		if assert.NoError(t, err) && assert.Len(t, out, 1) {
			assert.Equal(t, "registryuser", out[0].Username)
			assert.Equal(t, "registrypass", out[0].Password)
		}
	})

	t.Run("exact registry key wins over trailing-slash registry key", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"registry.gitlab.com\": {\"username\": \"exactuser\", \"password\": \"exactpass\"}, \"registry.gitlab.com/\": {\"username\": \"slashuser\", \"password\": \"slashpass\"}}}")},
		}

		// "registry.gitlab.com/" is the registry root, not a repo path, so it must
		// not be treated as a more-specific path match that beats the exact key.
		out, err := ParseSecretData([]corev1.Secret{secret}, "registry.gitlab.com", "registry.gitlab.com/company/image")

		assert.NoError(t, err)
		if assert.Len(t, out, 1) {
			assert.Equal(t, "exactuser", out[0].Username)
			assert.Equal(t, "exactpass", out[0].Password)
		}
	})

	t.Run("path-scoped credential wins over scheme-prefixed registry credential", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://registry.gitlab.com\": {\"username\": \"registryuser\", \"password\": \"registrypass\"}, \"registry.gitlab.com/company/path\": {\"username\": \"pathuser\", \"password\": \"pathpass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "registry.gitlab.com", "registry.gitlab.com/company/path/image")

		if assert.NoError(t, err) && assert.Len(t, out, 1) {
			assert.Equal(t, "pathuser", out[0].Username)
			assert.Equal(t, "pathpass", out[0].Password)
		}
	})

	t.Run("scheme-insensitive registry match prefers the secure https key", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://reg.example.com\": {\"username\": \"httpsuser\", \"password\": \"httpspass\"}, \"http://reg.example.com\": {\"username\": \"httpuser\", \"password\": \"httppass\"}}}")},
		}

		// When both an insecure "http://" and a secure "https://" key match the
		// same host, the https credential is chosen deterministically so we never
		// send credentials intended for the secure endpoint to an insecure one.
		for range 20 {
			out, err := ParseSecretData([]corev1.Secret{secret}, "reg.example.com", "reg.example.com/image")

			assert.NoError(t, err)
			if assert.Len(t, out, 1) {
				assert.Equal(t, "httpsuser", out[0].Username)
				assert.Equal(t, "httpspass", out[0].Password)
			}
		}
	})

	t.Run("repository namespace named v2 is treated as path-scoped, not registry-wide", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"myreg.io/v2\": {\"username\": \"v2user\", \"password\": \"v2pass\"}}}")},
		}

		// "myreg.io/v2" is a repo namespace, not the legacy "/v1/"-style registry
		// suffix, so its credentials must not be applied to an unrelated repo.
		out, err := ParseSecretData([]corev1.Secret{secret}, "myreg.io", "myreg.io/other/app")

		assert.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("legacy docker credential-store key matches bare registry host", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://index.docker.io/v1/\": {\"username\": \"dockeruser\", \"password\": \"dockerpass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "index.docker.io", "index.docker.io/library/nginx:latest")

		if assert.NoError(t, err) && assert.Len(t, out, 1) {
			assert.Equal(t, "dockeruser", out[0].Username)
			assert.Equal(t, "dockerpass", out[0].Password)
		}
	})

	t.Run("parse docker config missed data", func(t *testing.T) {

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dockercfg",
			},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "fake", "image")

		assert.Nil(t, out)
		assert.EqualError(t, err, "imagePullSecret dockercfg contains neither .dockercfg nor .dockerconfigjson")
	})

	t.Run("parse docker config wrong auth", func(t *testing.T) {

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://index.docker.io/v1/\": {\"auth\": \"12345\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "https://index.docker.io/v1/", "image")

		assert.Nil(t, out)
		assert.ErrorContains(t, err, "illegal base64 data")
	})

}
