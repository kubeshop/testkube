package agents

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewEnableAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent <name>",
		Aliases: []string{"runner", "gitops"},
		Args:    cobra.ExactArgs(1),
		Hidden:  !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			UiEnableAgent(cmd, strings.Join(args, ""))
		},
	}

	return cmd
}

func NewDisableAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent <name>",
		Aliases: []string{"runner", "gitops"},
		Args:    cobra.ExactArgs(1),
		Hidden:  !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			UiDisableAgent(cmd, strings.Join(args, ""))
		},
	}

	return cmd
}

func UiEnableAgent(cmd *cobra.Command, name string) {
	agent, err := GetControlPlaneAgent(cmd, name)
	ui.ExitOnError("getting agent", err)

	if agent.Disabled {
		agent, err = UpdateAgent(cmd, agent.ID, client.AgentInput{
			Disabled: common.Ptr(false),
		})
		ui.ExitOnError("updating agent", err)
	} else {
		fmt.Println("Agent is already enabled.")
	}

	PrintControlPlaneAgent(*agent)
}

func UiDisableAgent(cmd *cobra.Command, name string) {
	agent, err := GetControlPlaneAgent(cmd, name)
	ui.ExitOnError("getting agent", err)

	if !agent.Disabled {
		agent, err = UpdateAgent(cmd, agent.ID, client.AgentInput{
			Disabled: common.Ptr(true),
		})
		ui.ExitOnError("updating agent", err)
	} else {
		fmt.Println("Agent is already enabled.")
	}

	PrintControlPlaneAgent(*agent)
}
