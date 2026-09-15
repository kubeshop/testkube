package agents

import (
	"strings"

	"github.com/spf13/cobra"

	common2 "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewEnableAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "runner <name>",
		Aliases: []string{"agent", "gitops"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UiEnableAgent(cmd, strings.Join(args, ""))
		},
	}

	return cmd
}

func NewDisableAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "runner <name>",
		Aliases: []string{"agent", "gitops"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UiDisableAgent(cmd, strings.Join(args, ""))
		},
	}

	return cmd
}

func UiEnableAgent(cmd *cobra.Command, name string) {
	agent, err := GetControlPlaneAgent(cmd, name)
	if err != nil {
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrAgentGetFailed,
			"Error getting the runner",
			common2.AgentLookupHint,
			err,
		))
	}

	if agent.Disabled {
		agent, err = UpdateAgent(cmd, agent.ID, client.AgentInput{
			Disabled: common.Ptr(false),
		})
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrAgentWriteFailed,
				"Error enabling the runner",
				common2.AgentWriteHint,
				err,
			))
		}
	} else {
		ui.Print("Runner is already enabled.")
	}

	PrintControlPlaneAgent(*agent)
}

func UiDisableAgent(cmd *cobra.Command, name string) {
	agent, err := GetControlPlaneAgent(cmd, name)
	if err != nil {
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrAgentGetFailed,
			"Error getting the runner",
			common2.AgentLookupHint,
			err,
		))
	}

	if !agent.Disabled {
		agent, err = UpdateAgent(cmd, agent.ID, client.AgentInput{
			Disabled: common.Ptr(true),
		})
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrAgentWriteFailed,
				"Error disabling the runner",
				common2.AgentWriteHint,
				err,
			))
		}
	} else {
		ui.Print("Runner is already disabled.")
	}

	PrintControlPlaneAgent(*agent)
}
