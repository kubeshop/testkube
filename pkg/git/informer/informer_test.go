package informer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestNormalizePaths(t *testing.T) {
	paths := []string{" /a ", "/b/c", "", "///", "d/"}
	assert.Equal(t, []string{"a", "b/c", "d"}, normalizePaths(paths))
}

func TestResolveCredentialValue(t *testing.T) {
	t.Setenv("TK_GIT_USERNAME", "env-user")
	t.Setenv("TK_GIT_TOKEN", "env-token")

	assert.Equal(t, "inline", resolveCredentialValue("inline", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Key: "TK_GIT_USERNAME"},
	}))
	assert.Equal(t, "env-user", resolveCredentialValue("", &testkube.EnvVarSource{
		SecretKeyRef: &testkube.EnvVarSourceSecretKeyRef{Key: "TK_GIT_USERNAME"},
	}))
	assert.Equal(t, "env-token", resolveCredentialValue("", &testkube.EnvVarSource{
		ConfigMapKeyRef: &testkube.EnvVarSourceConfigMapKeyRef{Key: "TK_GIT_TOKEN"},
	}))
	assert.Equal(t, "", resolveCredentialValue("", nil))
}

func TestAuthClientOptions(t *testing.T) {
	t.Run("basic auth default", func(t *testing.T) {
		opts, err := authClientOptions(&testkube.TestTriggerContentGit{
			Username: "user",
			Token:    "token",
		})
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})

	t.Run("header auth", func(t *testing.T) {
		authType := testkube.HEADER_ContentGitAuthType
		opts, err := authClientOptions(&testkube.TestTriggerContentGit{
			Token:    "token",
			AuthType: &authType,
		})
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})

	t.Run("ssh auth", func(t *testing.T) {
		const testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAoL8Hj5vDi+FKsxlagvFY/oK879UYeXpn0AZL2ryHXvbzQxqj
hUP5rkgea5gwrXh/aU070I+G2xIELB3Pxkkn9kdNogkOClmcGz4kUtIzRyuQsQt9
d7TwhIURbQo8MKfGFIy4Bnb4XpjxcsISuFdOj4MP5JpEtxbDc1XkuVG7cb80entn
LeQwQXohWDAp3qCX6SR7e7jagc1pi2XEgLwkD1BpBkYTyZxGmAm07K1iU3+25zOd
+fa7PcFPjDPVRcJo7TjBkPrpUaZDhpfhRjTDEPaBXDKFD34jFeE2fxnsrhCCh9T/
p18vt9NyNsMTJ/m80u1ur4ymNqdj3zWRk5fdGwIDAQABAoIBAAvGyFsGYlvLyZk4
G6Ae5meJqdCNoBlmP6zHM/ohIxnF+ynIuH95l2qOm2FD6Re3DZPoC0sgThTxS5+f
yipmDv4ard6tIyYxonTXJ09vWZUL5Vx90ayvc4oskDZDhbKwLUbcIwSg5VlTgyeR
KFCbtNm2vwypxgGpMpsvM8PlRQJJjhLAM0++9Irrf0BfjmZOj4StzDHdvljDwHer
ljmGsfVSt/NR/14WQ3q0QvGMkP84pf8qWQIKvSuClzSiJ3G5J/rM0DSiZsv5/ags
loUVDsPsMVIz2nFerO0Me4aBeLzG+yih+bPqF+CQO2KUL3f0CAcf6uHEzcEohSGp
EMe7GyECgYEAzrZKZGceBVZEYW4/5Q6KFPnOcwaEzJtg7Av3jztREH0nYCkd8xO0
q/1MZYq2C/HcATk3f6xXvS0/DkUWqS92EmHsQWWjahlSl4sIBsh5BlneKZ+KQLmf
jv2LjnJoLuuAjpkJ1pssrWK22gO8JlJBSnbPUMh/eYEc39DFT4CqxjUCgYEAxxL7
TEmss5k9T89FkqEwAc1tKYiCB7k9buI+nja5clWyULMb5iB0bGogdz1PJEX341SI
le3f3qoUM/Q4UzGi6FrX5ItRfAlLH/Wc3Jsd2fPc2BJCgDNCaRl30h5Xx6n6znz8
qFe1C0FFeY0faeCNlGLyARd7P97Lg3FH0bUFQA8CgYA7Q03erSWROCNQn5AX9mwm
CVxj49mM43sNEX0/Bi1+gbMZQZCBkQO6T1tovTTmBcgiXaoIo3tgFCnAyJPvm1jJ
emOGeEI6d9oS8lwxvaXc6UTlQAUd+1nAX/Zzt18hHIl12HBWo5RSfTuZE3sMrYZk
d92F9oV9a0PA8xSub2AGhQKBgAVjgCXqgKBD76Lva2SytEf4NZJAPbTT0NPlj+hc
dtyfcTo5/vFVw5EDtmlD4ZaLxlADA8d7LuoqFG3rmHK4Dz7W5q0rEEOZRM1SqrJW
CJLTxRCcPeyWdp+9rr6jT6D5+u4H+BbeeOobFDRcG5OUHoD7xK0+43kxILUoJdeJ
XOEFAoGBALH/+nWluf9vQsYjyKNxpQMdVbf+eDTUJjTIPnQT1RYwbNMaTzWRhcxi
QjPEcZ4CgqOZ0uxWNT2DiuIpYMO/c+Gxe3Na30soqvuzTJ3IXiflaFOjVrvOym2f
x1YSAMuOHPoj7BMCm2SVFQKMTMFNMtsCRJ8XDhi5QsL/xJank/TL
-----END RSA PRIVATE KEY-----`

		opts, err := authClientOptions(&testkube.TestTriggerContentGit{
			SshKey: testPrivateKey,
		})
		require.NoError(t, err)
		assert.Len(t, opts, 1)

		_, err = authClientOptions(&testkube.TestTriggerContentGit{
			SshKey: "invalid-private-key",
		})
		require.Error(t, err)
	})
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
			name: "valid git content trigger via resourceRef",
			trigger: testkube.TestTrigger{
				ResourceRef: &testkube.TestTriggerResourceRef{
					Kind: "Content",
				},
				ContentSelector: &testkube.TestTriggerContentSelector{
					Git: &testkube.TestTriggerContentGit{
						Uri: "https://github.com/example/repo.git",
					},
				},
			},
			expected: true,
		},
		{
			name: "resourceRef non-content",
			trigger: testkube.TestTrigger{
				ResourceRef: &testkube.TestTriggerResourceRef{
					Kind: "Deployment",
				},
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
