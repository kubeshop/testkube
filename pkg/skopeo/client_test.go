package skopeo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseSecretData(t *testing.T) {
	t.Parallel()

	t.Run("parse docker config base auth", func(t *testing.T) {
		t.Parallel()

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://index.docker.io/v1/\": {\"auth\": \"ZG9ja2VyLXVzZXJuYW1lOnlvdXItcmVhbGx5LXJlYWxseS1sb25nLWF1dGgta2V5\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "https://index.docker.io/v1/")

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "docker-username", out[0].Username)
		assert.Equal(t, "your-really-really-long-auth-key", out[0].Password)
		assert.NoError(t, err)
	})

	t.Run("parse docker config map", func(t *testing.T) {
		t.Parallel()

		secret := corev1.Secret{
			Data: map[string][]byte{".dockercfg": []byte("{\"https://index.docker.io/v1/\": {\"auth\": \"ZG9ja2VyLXVzZXJuYW1lOnlvdXItcmVhbGx5LXJlYWxseS1sb25nLWF1dGgta2V5\"}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "https://index.docker.io/v1/")

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "docker-username", out[0].Username)
		assert.Equal(t, "your-really-really-long-auth-key", out[0].Password)
		assert.NoError(t, err)
	})

	t.Run("parse docker config plain credentials", func(t *testing.T) {
		t.Parallel()

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://index.docker.io/v1/\": {\"username\": \"plainuser\", \"password\": \"plainpass\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "https://index.docker.io/v1/")

		assert.Equal(t, 1, len(out))
		assert.Equal(t, "plainuser", out[0].Username)
		assert.Equal(t, "plainpass", out[0].Password)
		assert.NoError(t, err)
	})

	t.Run("parse docker config missed data", func(t *testing.T) {
		t.Parallel()

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dockercfg",
			},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "fake")

		assert.Nil(t, out)
		assert.EqualError(t, err, "imagePullSecret dockercfg contains neither .dockercfg nor .dockerconfigjson")
	})

	t.Run("parse docker config wrong auth", func(t *testing.T) {
		t.Parallel()

		secret := corev1.Secret{
			Data: map[string][]byte{".dockerconfigjson": []byte("{\"auths\": {\"https://index.docker.io/v1/\": {\"auth\": \"12345\"}}}")},
		}

		out, err := ParseSecretData([]corev1.Secret{secret}, "https://index.docker.io/v1/")

		assert.Nil(t, out)
		assert.ErrorContains(t, err, "illegal base64 data")
	})

}

// TestTrimTopNonJSON tests the trimNonJSON function with various inputs to ensure it correctly trims non-JSON leading characters.
func TestTrimTopNonJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "No JSON",
			input:    []byte("hello world"),
			expected: nil,
		},
		{
			name:     "Valid JSON at start",
			input:    []byte(`{"key": "value"}`),
			expected: []byte(`{"key": "value"}`),
		},
		{
			name:     "JSON with leading text",
			input:    []byte(`error: failed {"key": "value"}`),
			expected: []byte(`{"key": "value"}`),
		},
		{
			name:     "Multiple JSON objects, trim to first",
			input:    []byte(`error: failed {"key1": "value1"} another error {"key2": "value2"}`),
			expected: []byte(`{"key1": "value1"} another error {"key2": "value2"}`),
		},
		{
			name:     "No opening brace",
			input:    []byte(`error: failed no json here`),
			expected: nil,
		},
		{
			name: "Leading spaces and newline before JSON",
			input: []byte(`  
			            {"key": "value"}`),
			expected: []byte(`{"key": "value"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimTopNonJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
