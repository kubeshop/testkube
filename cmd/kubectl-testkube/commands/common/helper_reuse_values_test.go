package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

func upgradeOptions() HelmOptions {
	// the defaults of `testkube upgrade`
	return HelmOptions{Name: "testkube", Namespace: "testkube", Chart: "kubeshop/testkube", NoPostgres: true}
}

func TestPrepareTestkubeHelmArgs_ReuseValues(t *testing.T) {
	t.Run("sends every setting without --reuse-values", func(t *testing.T) {
		args := prepareTestkubeHelmArgs(upgradeOptions())

		assert.NotContains(t, args, "--reuse-values")
		assert.Contains(t, args, "postgresql.enabled=false")
		assert.Contains(t, args, "mongodb.enabled=true")
		assert.Contains(t, args, "testkube-api.logs.storage=minio")
	})

	t.Run("keeps the release's values for the flags not passed", func(t *testing.T) {
		options := upgradeOptions()
		options.ReuseValues = true

		args := prepareTestkubeHelmArgs(options)

		assert.Contains(t, args, "--reuse-values")
		assert.NotContains(t, args, "--set", "a flag default would overwrite a value of the release")
	})

	t.Run("overrides the values of the flags passed", func(t *testing.T) {
		options := upgradeOptions()
		options.ReuseValues = true
		options.NoPostgres = false
		options.ExplicitFlags = map[string]bool{"no-postgres": true}

		args := prepareTestkubeHelmArgs(options)

		assert.Contains(t, args, "--reuse-values")
		assert.Contains(t, args, "postgresql.enabled=true")
		assert.NotContains(t, args, "mongodb.enabled=true")
	})

	t.Run("keeps the --set options of the user", func(t *testing.T) {
		options := upgradeOptions()
		options.ReuseValues = true
		options.SetOptions = map[string]string{"testkube-api.replicas": "2"}

		args := prepareTestkubeHelmArgs(options)

		assert.Contains(t, args, "testkube-api.replicas=2")
	})
}

func TestPrepareTestkubeProHelmArgs_ReuseValues(t *testing.T) {
	options := upgradeOptions()
	options.ReuseValues = true
	options.NoMinio, options.NoMongo, options.NoPostgres = true, true, true
	options.Master = config.Master{AgentToken: "tkcagnt_key", URIs: config.MasterURIs{Agent: "agent.testkube.io:443"}}

	args := prepareTestkubeProHelmArgs(options, false)

	assert.Contains(t, args, "--reuse-values")
	assert.Contains(t, args, "testkube-api.minio.enabled=false", "a runner disables its storage on purpose")
	assert.Contains(t, args, "mongodb.enabled=false")
	assert.Contains(t, args, "postgresql.enabled=false")
	assert.Contains(t, args, "testkube-api.cloud.key=tkcagnt_key")
	assert.Contains(t, args, "testkube-api.cloud.url=agent.testkube.io:443")
	assert.NotContains(t, args, "testkube-api.cloud.uiUrl=", "an unknown value must not clear the release's")
}
