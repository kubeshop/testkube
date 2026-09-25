package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/agent"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/agents"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/debug"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/docker"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/pro"
)

func TestRunnerCommandAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     *cobra.Command
		primary string
		aliases []string
	}{
		{name: "top-level runner", cmd: NewAgentCmd(), primary: "runner", aliases: []string{"agent"}},
		{name: "create runner", cmd: agents.NewCreateAgentCommand(), primary: "runner", aliases: []string{"agent"}},
		{name: "install runner", cmd: agents.NewInstallAgentCommand(), primary: "runner", aliases: []string{"agent"}},
		{name: "get runner", cmd: agents.NewGetAgentCommand(), primary: "runner", aliases: []string{"runners", "agent", "agents", "a"}},
		{name: "delete runner", cmd: agents.NewDeleteAgentCommand(), primary: "runner", aliases: []string{"agent"}},
		{name: "update runner", cmd: agents.NewUpdateAgentCommand(), primary: "runner", aliases: []string{"agent"}},
		{name: "enable runner", cmd: agents.NewEnableAgentCommand(), primary: "runner", aliases: []string{"agent", "gitops"}},
		{name: "disable runner", cmd: agents.NewDisableAgentCommand(), primary: "runner", aliases: []string{"agent", "gitops"}},
		{name: "debug runner", cmd: debug.NewDebugAgentCmd(), primary: "runner", aliases: []string{"agent", "ag", "a"}},
		{name: "migrate runner", cmd: agent.NewMigrateAgentCmd(), primary: "runner", aliases: []string{"agent"}},
		{name: "init runner", cmd: pro.NewInitCmd(), primary: "runner", aliases: []string{"install", "agent", "init"}},
		{name: "docker init", cmd: docker.NewInitCmd(), primary: "init", aliases: []string{"install", "agent", "runner"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertCommandResolves(t, tt.cmd, tt.primary, tt.aliases...)
		})
	}
}

func TestInitProfileAliases(t *testing.T) {
	t.Parallel()

	initCmd := NewInitCmd()
	assertCommandResolves(t, findSubcommand(t, initCmd, "standalone-runner"), "standalone-runner", "oss", "standalone", "standalone-agent")
	assertCommandResolves(t, findSubcommand(t, initCmd, "runner"), "runner", "agent", "install", "init")
}

func findSubcommand(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()
	cmd, _, err := parent.Find([]string{name})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	return cmd
}

func assertCommandResolves(t *testing.T, cmd *cobra.Command, primary string, aliases ...string) {
	t.Helper()
	require.Equal(t, primary, cmd.Name())

	parent := &cobra.Command{Use: "parent"}
	parent.AddCommand(cmd)

	got, _, err := parent.Find([]string{primary})
	require.NoError(t, err)
	assert.Equal(t, cmd, got)

	for _, alias := range aliases {
		got, _, err = parent.Find([]string{alias})
		require.NoError(t, err, "alias %q should resolve", alias)
		assert.Equal(t, cmd, got, "alias %q should resolve to %s", alias, primary)
	}
}
