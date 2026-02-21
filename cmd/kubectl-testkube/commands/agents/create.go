package agents

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateAgentCommand() *cobra.Command {
	var (
		labelPairs     []string
		environmentIds []string
		global         bool
		group          string
		floating       bool
		agentType      string
	)
	cmd := &cobra.Command{
		Use:  "agent",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check for deprecated --type flag usage
			if cmd.Flags().Changed("type") {
				ui.Warn("⚠️  The --type/-t flag is deprecated.")
				ui.Info("Please use capability flags instead:")
				ui.Info("  --runner    : Enable runner capability")
				ui.Info("  --listener  : Enable listener capability")
				ui.Info("  --gitops    : Enable GitOps capability")
				ui.Info("  --webhooks  : Enable webhooks capability")
				ui.NL()
				return
			}

			runnerChanged := cmd.Flags().Changed("runner")
			listenerChanged := cmd.Flags().Changed("listener")
			gitopsChanged := cmd.Flags().Changed("gitops")
			webhooksChanged := cmd.Flags().Changed("webhooks")
			anyChanged := runnerChanged || listenerChanged || gitopsChanged || webhooksChanged
			enableRunner, _ := cmd.Flags().GetBool("runner")
			enableListener, _ := cmd.Flags().GetBool("listener")
			enableGitops, _ := cmd.Flags().GetBool("gitops")
			enableWebhooks, _ := cmd.Flags().GetBool("webhooks")
			// we default to both capabilities if none flags are set
			if !anyChanged {
				enableRunner = true
				enableListener = true
			}

			agent := UiCreateAgent(
				cmd,
				args[0],
				labelPairs,
				environmentIds,
				global,
				group,
				floating,
				enableRunner,
				enableListener,
				enableGitops,
				enableWebhooks,
			)
			ui.NL()
			ui.Info("Install the agent with command:")
			installCmd := fmt.Sprintf("kubectl testkube install agent %s --secret %s", agent.Name, agent.SecretKey)
			if enableRunner {
				installCmd += " --runner"
			}
			if enableListener {
				installCmd += " --listener"
			}
			if enableGitops {
				installCmd += " --gitops"
			}
			if enableWebhooks {
				installCmd += " --webhooks"
			}
			ui.ShellCommand(installCmd)
		},
	}

	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&global, "global", false, "make it global agent")
	cmd.Flags().StringVar(&group, "group", "", "make it grouped agent")
	cmd.Flags().BoolVar(&floating, "floating", false, "create as a floating agent")

	// Components selection
	cmd.Flags().Bool("runner", false, "enable runner capability (default: enabled when no component flags are set)")
	cmd.Flags().Bool("listener", false, "enable listener capability (default: enabled when no component flags are set)")
	cmd.Flags().Bool("gitops", false, "enable gitops capability")
	cmd.Flags().Bool("webhooks", false, "enable webhooks capability")

	// Deprecated flag
	cmd.Flags().StringVarP(&agentType, "type", "t", "", "[DEPRECATED] agent type - use capability flags instead")
	cmd.Flags().MarkDeprecated("type", "use --runner, --listener, --gitops, and/or --webhooks instead")

	return cmd
}

// NewCreateRunnerCommand creates an agent with runner-only capability (no listener).
func NewCreateRunnerCommand() *cobra.Command {
	var (
		labelPairs     []string
		environmentIds []string
		global         bool
		group          string
		floating       bool
		agentType      string
	)

	cmd := &cobra.Command{
		Use:  "runner",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check for deprecated --type flag usage
			if cmd.Flags().Changed("type") {
				ui.Warn("⚠️  The --type/-t flag is deprecated.")
				ui.Info("This command creates a runner-only agent by default.")
				ui.Info("For more flexibility, use 'kubectl testkube create agent' with:")
				ui.Info("  --runner    : Enable runner capability")
				ui.Info("  --listener  : Enable listener capability")
				ui.Info("  --gitops    : Enable GitOps capability")
				ui.Info("  --webhooks  : Enable webhooks capability")
				ui.NL()
			}

			agent := UiCreateAgent(cmd, args[0], labelPairs, environmentIds, global, group, floating, true, false, false, false)
			ui.NL()
			ui.Info("Install the agent with command:")
			installCmd := fmt.Sprintf("kubectl testkube install agent %s --secret %s --runner", agent.Name, agent.SecretKey)
			ui.ShellCommand(installCmd)
		},
	}

	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&global, "global", false, "make it global agent")
	cmd.Flags().StringVar(&group, "group", "", "make it grouped agent")
	cmd.Flags().BoolVar(&floating, "floating", false, "create as a floating agent")

	// Deprecated flag
	cmd.Flags().StringVarP(&agentType, "type", "t", "", "[DEPRECATED] agent type - this command creates runner-only agents")
	cmd.Flags().MarkDeprecated("type", "this command creates runner-only agents by default")

	return cmd
}
