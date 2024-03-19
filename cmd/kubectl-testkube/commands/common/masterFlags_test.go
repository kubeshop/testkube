package common

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
)

var (
	opts = HelmOptions{}
	cfg  = &config.Data{}
)

func NewTestCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:    "test",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			ProcessMasterFlags(cmd, &opts, cfg)
		},
	}

	PopulateMasterFlags(cmd, &opts)
	PopulateHelmFlags(cmd, &opts)
	return cmd
}

func TestMasterCmds(t *testing.T) {
	t.Run("Test all master flags set and isnecure", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--master-insecure", "true",
			"--agent-prefix", "dummy-agent-prefix",
			"--api-prefix", "dummy-api-prefix",
			"--ui-prefix", "dummy-ui-prefix",
			"--root-domain", "dummy-root-domain",
			"--dry-run", "true",
			"--no-confirm", "true",
			"--agent-token", "dummy-token",
			"--agent-uri", "dummy-uri",
			"--ui-prefix", "dummy-ui-prefix",
		})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, true, opts.Master.Insecure)
		assert.Equal(t, true, opts.DryRun)
		assert.Equal(t, true, opts.NoConfirm)
		assert.Equal(t, "dummy-token", opts.Master.AgentToken)
		assert.Equal(t, "dummy-uri", opts.Master.URIs.Agent)
		assert.Equal(t, "dummy-ui-prefix", opts.Master.UiUrlPrefix)
		assert.Equal(t, "dummy-api-prefix", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "dummy-agent-prefix", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "dummy-root-domain", opts.Master.RootDomain)
		assert.Equal(t, "http://dummy-api-prefix.dummy-root-domain", opts.Master.URIs.Api)
		assert.Equal(t, "http://dummy-ui-prefix.dummy-root-domain", opts.Master.URIs.Ui)
	})
	t.Run("Test all master flags set and secure", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--agent-prefix", "dummy-agent-prefix",
			"--api-prefix", "dummy-api-prefix",
			"--ui-prefix", "dummy-ui-prefix",
			"--root-domain", "dummy-root-domain",
			"--dry-run", "true",
			"--no-confirm", "true",
			"--agent-token", "dummy-token",
			"--agent-uri", "dummy-uri",
			"--ui-prefix", "dummy-ui-prefix",
		})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, false, opts.Master.Insecure)
		assert.Equal(t, true, opts.DryRun)
		assert.Equal(t, true, opts.NoConfirm)
		assert.Equal(t, "dummy-token", opts.Master.AgentToken)
		assert.Equal(t, "dummy-uri", opts.Master.URIs.Agent)
		assert.Equal(t, "dummy-ui-prefix", opts.Master.UiUrlPrefix)
		assert.Equal(t, "dummy-api-prefix", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "dummy-agent-prefix", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "dummy-root-domain", opts.Master.RootDomain)
		assert.Equal(t, "https://dummy-api-prefix.dummy-root-domain", opts.Master.URIs.Api)
		assert.Equal(t, "https://dummy-ui-prefix.dummy-root-domain", opts.Master.URIs.Ui)
	})

	t.Run("Test all master flags set and isnecure former cloud flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--cloud-insecure", "true",
			"--cloud-agent-prefix", "dummy-agent-prefix",
			"--cloud-api-prefix", "dummy-api-prefix",
			"--cloud-ui-prefix", "dummy-ui-prefix",
			"--cloud-root-domain", "dummy-root-domain",
			"--dry-run", "true",
			"--no-confirm", "true",
			"--agent-token", "dummy-token",
			"--agent-uri", "dummy-uri",
			"--ui-prefix", "dummy-ui-prefix",
		})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, true, opts.Master.Insecure)
		assert.Equal(t, true, opts.DryRun)
		assert.Equal(t, true, opts.NoConfirm)
		assert.Equal(t, "dummy-token", opts.Master.AgentToken)
		assert.Equal(t, "dummy-uri", opts.Master.URIs.Agent)
		assert.Equal(t, "dummy-ui-prefix", opts.Master.UiUrlPrefix)
		assert.Equal(t, "dummy-api-prefix", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "dummy-agent-prefix", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "dummy-root-domain", opts.Master.RootDomain)
		assert.Equal(t, "http://dummy-api-prefix.dummy-root-domain", opts.Master.URIs.Api)
		assert.Equal(t, "http://dummy-ui-prefix.dummy-root-domain", opts.Master.URIs.Ui)
	})
	t.Run("Test all master flags set and secure former cloud flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--cloud-agent-prefix", "dummy-agent-prefix",
			"--cloud-api-prefix", "dummy-api-prefix",
			"--cloud-ui-prefix", "dummy-ui-prefix",
			"--cloud-root-domain", "dummy-root-domain",
			"--dry-run", "true",
			"--no-confirm", "true",
			"--agent-token", "dummy-token",
			"--agent-uri", "dummy-uri",
			"--ui-prefix", "dummy-ui-prefix",
		})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, false, opts.Master.Insecure)
		assert.Equal(t, true, opts.DryRun)
		assert.Equal(t, true, opts.NoConfirm)
		assert.Equal(t, "dummy-token", opts.Master.AgentToken)
		assert.Equal(t, "dummy-uri", opts.Master.URIs.Agent)
		assert.Equal(t, "dummy-ui-prefix", opts.Master.UiUrlPrefix)
		assert.Equal(t, "dummy-api-prefix", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "dummy-agent-prefix", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "dummy-root-domain", opts.Master.RootDomain)
		assert.Equal(t, "https://dummy-api-prefix.dummy-root-domain", opts.Master.URIs.Api)
		assert.Equal(t, "https://dummy-ui-prefix.dummy-root-domain", opts.Master.URIs.Ui)
	})

	t.Run("Test defaults for master flags secure", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, false, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "testkube.io", opts.Master.RootDomain)
		assert.Equal(t, "https://api.testkube.io", opts.Master.URIs.Api)
		assert.Equal(t, "https://app.testkube.io", opts.Master.URIs.Ui)
		assert.Equal(t, "agent.testkube.io:443", opts.Master.URIs.Agent)
	})

	t.Run("Test defaults for master flags insecure", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--master-insecure", "true"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, true, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "testkube.io", opts.Master.RootDomain)
		assert.Equal(t, "http://api.testkube.io", opts.Master.URIs.Api)
		assert.Equal(t, "http://app.testkube.io", opts.Master.URIs.Ui)
		assert.Equal(t, "agent.testkube.io:443", opts.Master.URIs.Agent)
	})

	t.Run("Test defaults for master flags insecure former cloud flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--cloud-insecure", "true"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, true, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "testkube.io", opts.Master.RootDomain)
		assert.Equal(t, "http://api.testkube.io", opts.Master.URIs.Api)
		assert.Equal(t, "http://app.testkube.io", opts.Master.URIs.Ui)
		assert.Equal(t, "agent.testkube.io:443", opts.Master.URIs.Agent)
	})

	t.Run("Test defaults for master flags secure with rood modified", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--root-domain", "dummy-root-domain"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, false, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "dummy-root-domain", opts.Master.RootDomain)
		assert.Equal(t, "https://api.dummy-root-domain", opts.Master.URIs.Api)
		assert.Equal(t, "https://app.dummy-root-domain", opts.Master.URIs.Ui)
		assert.Equal(t, "agent.dummy-root-domain:443", opts.Master.URIs.Agent)
	})

	t.Run("Test defaults for master flags insecure with rood modified", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--master-insecure", "true", "--root-domain", "dummy-root-domain"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, true, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "dummy-root-domain", opts.Master.RootDomain)
		assert.Equal(t, "http://api.dummy-root-domain", opts.Master.URIs.Api)
		assert.Equal(t, "http://app.dummy-root-domain", opts.Master.URIs.Ui)
		assert.Equal(t, "agent.dummy-root-domain:443", opts.Master.URIs.Agent)
	})

	t.Run("Test defaults for master flags secure with root modifiedc former cloud flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--cloud-root-domain", "dummy-root-domain"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, false, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "dummy-root-domain", opts.Master.RootDomain)
		assert.Equal(t, "https://api.dummy-root-domain", opts.Master.URIs.Api)
		assert.Equal(t, "https://app.dummy-root-domain", opts.Master.URIs.Ui)
		assert.Equal(t, "agent.dummy-root-domain:443", opts.Master.URIs.Agent)
	})

	t.Run("Test defaults for master flags insecure with root modified former cloud flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--cloud-insecure", "true", "--cloud-root-domain", "dummy-root-domain"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, true, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "dummy-root-domain", opts.Master.RootDomain)
		assert.Equal(t, "http://api.dummy-root-domain", opts.Master.URIs.Api)
		assert.Equal(t, "http://app.dummy-root-domain", opts.Master.URIs.Ui)
		assert.Equal(t, "agent.dummy-root-domain:443", opts.Master.URIs.Agent)
	})

	t.Run("Test defaults for master flags secure with agent uri modified", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--agent-uri", "dummy-agent-uri"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, false, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "testkube.io", opts.Master.RootDomain)
		assert.Equal(t, "https://api.testkube.io", opts.Master.URIs.Api)
		assert.Equal(t, "https://app.testkube.io", opts.Master.URIs.Ui)
		assert.Equal(t, "dummy-agent-uri", opts.Master.URIs.Agent)
	})

	t.Run("Test defaults for master flags insecure with agent uri modified", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--master-insecure", "true", "--agent-uri", "dummy-agent-uri"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, true, opts.Master.Insecure)
		assert.Equal(t, false, opts.DryRun)
		assert.Equal(t, false, opts.NoConfirm)
		assert.Equal(t, "", opts.Master.AgentToken)
		assert.Equal(t, "app", opts.Master.UiUrlPrefix)
		assert.Equal(t, "api", opts.Master.ApiUrlPrefix)
		assert.Equal(t, "agent", opts.Master.AgentUrlPrefix)
		assert.Equal(t, "testkube.io", opts.Master.RootDomain)
		assert.Equal(t, "http://api.testkube.io", opts.Master.URIs.Api)
		assert.Equal(t, "http://app.testkube.io", opts.Master.URIs.Ui)
		assert.Equal(t, "dummy-agent-uri", opts.Master.URIs.Agent)
	})

	t.Run("Test --feature-logs-v2 feature disabled by default", func(t *testing.T) {
		cmd := NewTestCmd()
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, false, opts.Master.Features.LogsV2)
	})

	t.Run("Test --feature-logs-v2 feature flag", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--feature-logs-v2", "true"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, true, opts.Master.Features.LogsV2)
	})

	t.Run("Test --root-domain for master flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--root-domain", "test-domain"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, "test-domain", opts.Master.RootDomain)
	})

	t.Run("Test deprecated --cloud-root-domain for master flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--cloud-root-domain", "test-domain"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, "test-domain", opts.Master.RootDomain)
	})

	t.Run("Test deprecated --pro-root-domain for master flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--pro-root-domain", "test-domain"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, "test-domain", opts.Master.RootDomain)
	})

	t.Run("Test deprecated --cloud-root-domain and --pro-root-domain for master flags", func(t *testing.T) {
		cmd := NewTestCmd()
		cmd.SetArgs([]string{"--pro-root-domain", "pro-test-domain", "--cloud-root-domain", "cloud-test-domain"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, "pro-test-domain", opts.Master.RootDomain)
	})
}
