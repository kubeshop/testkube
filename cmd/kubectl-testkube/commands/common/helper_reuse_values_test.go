package common

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	runnerOptions := func(explicit ...string) HelmOptions {
		options := upgradeOptions()
		options.ReuseValues = true
		options.NoMinio, options.NoMongo, options.NoPostgres = true, true, true
		// ProcessMasterFlags always composes the addresses, from the default prefixes and
		// root domain when no flag is passed
		options.Master = config.Master{
			AgentToken: "tkcagnt_key",
			URIs:       config.MasterURIs{Agent: "agent.testkube.io:443", Ui: "https://app.testkube.io"},
		}
		options.ExplicitFlags = map[string]bool{}
		for _, name := range explicit {
			options.ExplicitFlags[name] = true
		}
		return options
	}

	t.Run("keeps the control plane of the release when no endpoint flag is passed", func(t *testing.T) {
		args := prepareTestkubeProHelmArgs(runnerOptions(), false)

		assert.Contains(t, args, "--reuse-values")
		assert.Contains(t, args, "testkube-api.minio.enabled=false", "a runner disables its storage on purpose")
		assert.Contains(t, args, "mongodb.enabled=false")
		assert.Contains(t, args, "postgresql.enabled=false")
		for _, arg := range args {
			assert.NotContains(t, arg, "testkube-api.cloud.", "a default address would disconnect a runner of a custom control plane")
		}
	})

	t.Run("sends the addresses composed from a passed root domain", func(t *testing.T) {
		args := prepareTestkubeProHelmArgs(runnerOptions("root-domain"), false)

		assert.Contains(t, args, "testkube-api.cloud.url=agent.testkube.io:443")
		assert.Contains(t, args, "testkube-api.cloud.uiUrl=https://app.testkube.io")
		assert.NotContains(t, args, "testkube-api.cloud.key=tkcagnt_key")
	})

	t.Run("sends only the runner address for --runner-uri", func(t *testing.T) {
		args := prepareTestkubeProHelmArgs(runnerOptions("runner-uri"), false)

		assert.Contains(t, args, "testkube-api.cloud.url=agent.testkube.io:443")
		assert.NotContains(t, args, "testkube-api.cloud.uiUrl=https://app.testkube.io")
	})

	t.Run("sends the key for --runner-token", func(t *testing.T) {
		args := prepareTestkubeProHelmArgs(runnerOptions("runner-token"), false)

		assert.Contains(t, args, "testkube-api.cloud.key=tkcagnt_key")
	})

	t.Run("sends everything without --reuse-values", func(t *testing.T) {
		options := runnerOptions()
		options.ReuseValues = false

		args := prepareTestkubeProHelmArgs(options, false)

		assert.NotContains(t, args, "--reuse-values")
		assert.Contains(t, args, "testkube-api.cloud.url=agent.testkube.io:443")
		assert.Contains(t, args, "testkube-api.cloud.key=tkcagnt_key")
		assert.Contains(t, args, "testkube-api.cloud.uiUrl=https://app.testkube.io")
	})
}

// Runs the flags through ProcessMasterFlags, so that every flag the composed addresses honour
// is checked, overrides included.
func TestPrepareTestkubeProHelmArgs_ReuseValuesWithParsedFlags(t *testing.T) {
	argsFor := func(t *testing.T, flags ...string) []string {
		options := upgradeOptions()
		cmd := &cobra.Command{Use: "upgrade"}
		PopulateMasterFlags(cmd, &options, false)
		require.NoError(t, cmd.ParseFlags(flags))
		ProcessMasterFlags(cmd, &options, nil)
		options.ReuseValues = true
		options.NoMinio, options.NoMongo, options.NoPostgres = true, true, true
		options.ExplicitFlags = map[string]bool{}
		cmd.Flags().Visit(func(f *pflag.Flag) { options.ExplicitFlags[f.Name] = true })
		return prepareTestkubeProHelmArgs(options, false)
	}

	tests := map[string]struct {
		flags  []string
		want   []string
		unsent []string
	}{
		"no endpoint flag": {
			unsent: []string{"testkube-api.cloud.url", "testkube-api.cloud.uiUrl", "testkube-api.cloud.key"},
		},
		"--runner-uri-override": {
			flags:  []string{"--runner-uri-override", "agent.example.com:443"},
			want:   []string{"testkube-api.cloud.url=agent.example.com:443"},
			unsent: []string{"testkube-api.cloud.uiUrl"},
		},
		"--agent-uri-override": {
			flags: []string{"--agent-uri-override", "agent.example.com:443"},
			want:  []string{"testkube-api.cloud.url=agent.example.com:443"},
		},
		"--ui-uri-override": {
			flags:  []string{"--ui-uri-override", "https://ui.example.com"},
			want:   []string{"testkube-api.cloud.uiUrl=https://ui.example.com"},
			unsent: []string{"testkube-api.cloud.url"},
		},
		"--root-domain": {
			flags: []string{"--root-domain", "example.com"},
			want:  []string{"testkube-api.cloud.url=agent.example.com:443", "testkube-api.cloud.uiUrl=https://app.example.com"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			args := argsFor(t, tt.flags...)

			for _, want := range tt.want {
				assert.Contains(t, args, want)
			}
			for _, unsent := range tt.unsent {
				for _, arg := range args {
					assert.NotContains(t, arg, unsent+"=")
				}
			}
		})
	}
}
