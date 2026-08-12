package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GenerateDemoAgentSecretKey(t *testing.T) {
	t.Parallel()

	t.Run("has the correct tkckey_<type>_<32 chars> format", func(t *testing.T) {
		t.Parallel()

		key := GenerateDemoAgentSecretKey()

		parts := strings.Split(key, "_")
		assert.Len(t, parts, 3, "key should have three underscore-separated parts")
		assert.Equal(t, "tkckey", parts[0], "key should start with the tkckey prefix")
		assert.Equal(t, "agent", parts[1], "key type should be agent")
		assert.Len(t, parts[2], 32, "random segment should be 32 characters")

		for _, r := range parts[2] {
			isLower := r >= 'a' && r <= 'z'
			isDigit := r >= '0' && r <= '9'
			assert.True(t, isLower || isDigit, "random segment must be lowercase alphanumeric, got %q", string(r))
		}
	})

	t.Run("produces a unique key per call", func(t *testing.T) {
		t.Parallel()

		const iterations = 100
		seen := make(map[string]struct{}, iterations)
		for i := 0; i < iterations; i++ {
			key := GenerateDemoAgentSecretKey()
			_, exists := seen[key]
			assert.False(t, exists, "generated a duplicate key: %s", key)
			seen[key] = struct{}{}
		}
	})
}

func Test_demoRunnerHelmOptions(t *testing.T) {
	t.Parallel()

	const (
		namespace = "testkube"
		secretKey = "tkckey_agent_abcdefghijklmnopqrstuvwxyz012345"
	)

	opts := demoRunnerHelmOptions(namespace, secretKey, false)

	t.Run("uses the published kubeshop/testkube-runner chart", func(t *testing.T) {
		assert.Equal(t, "https://kubeshop.github.io/helm-charts", opts.RegistryURL)
		assert.Equal(t, "kubeshop", opts.RepositoryName)
		assert.Equal(t, "testkube-runner", opts.ChartName)
		assert.Equal(t, "testkube-"+demoRunnerName, opts.ReleaseName)
		assert.Equal(t, namespace, opts.Namespace)
		assert.False(t, opts.DryRun)
		assert.Equal(t, []string{"--wait"}, opts.Args)
	})

	t.Run("wires the runner identity to match the bootstrapped runner", func(t *testing.T) {
		assert.Equal(t, demoRunnerID, opts.Values["runner.id"])
		assert.Equal(t, demoRunnerName, opts.Values["runner.name"])
		assert.Equal(t, demoRunnerOrgID, opts.Values["runner.orgId"])
		assert.Equal(t, demoRunnerEnvID, opts.Values["runner.envId"])
	})

	t.Run("passes through the generated secret key", func(t *testing.T) {
		assert.Equal(t, secretKey, opts.Values["runner.secret"])
	})

	t.Run("connects to the in-cluster control plane over plaintext gRPC", func(t *testing.T) {
		assert.Equal(t,
			"testkube-enterprise-api.testkube.svc.cluster.local:8089",
			opts.Values["cloud.url"],
		)
		assert.Equal(t, false, opts.Values["cloud.tls.enabled"])
	})

	t.Run("advertises the listener capability", func(t *testing.T) {
		assert.Equal(t, true, opts.Values["listener.enabled"])
	})

	t.Run("pins the deployment name via fullnameOverride", func(t *testing.T) {
		assert.Equal(t, "testkube-demo-runner", opts.Values["fullnameOverride"])
	})

	t.Run("interpolates the namespace into the cloud url", func(t *testing.T) {
		other := demoRunnerHelmOptions("custom-ns", secretKey, true)
		assert.Equal(t,
			"testkube-enterprise-api.custom-ns.svc.cluster.local:8089",
			other.Values["cloud.url"],
		)
		assert.True(t, other.DryRun)
	})
}

func Test_decideDemoAgentSecretKey(t *testing.T) {
	t.Parallel()

	const (
		namespace   = "testkube"
		existingKey = "tkckey_agent_abcdefghijklmnopqrstuvwxyz012345"
	)

	t.Run("reuses the existing runner key on re-install", func(t *testing.T) {
		t.Parallel()

		reuseKey, generate, cliErr := decideDemoAgentSecretKey(true, existingKey, false, namespace)
		assert.Nil(t, cliErr)
		assert.False(t, generate, "should not mint a new key when one can be reused")
		assert.Equal(t, existingKey, reuseKey)
	})

	t.Run("mints a fresh key on a clean namespace", func(t *testing.T) {
		t.Parallel()

		reuseKey, generate, cliErr := decideDemoAgentSecretKey(false, "", false, namespace)
		assert.Nil(t, cliErr)
		assert.True(t, generate, "should mint a new key when nothing is installed")
		assert.Equal(t, "", reuseKey)
	})

	t.Run("errors when a runner exists but its key is unreadable", func(t *testing.T) {
		t.Parallel()

		reuseKey, generate, cliErr := decideDemoAgentSecretKey(true, "", false, namespace)
		assert.NotNil(t, cliErr, "must not silently mint a mismatching key")
		assert.False(t, generate)
		assert.Equal(t, "", reuseKey)
	})

	t.Run("errors when the control plane exists but no runner to reuse from", func(t *testing.T) {
		t.Parallel()

		reuseKey, generate, cliErr := decideDemoAgentSecretKey(false, "", true, namespace)
		assert.NotNil(t, cliErr, "a fresh key would be rejected by the existing control plane")
		assert.False(t, generate)
		assert.Equal(t, "", reuseKey)
	})
}
