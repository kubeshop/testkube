package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/agents"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "install",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		}}

	cmd.AddCommand(agents.NewInstallAgentCommand())
	cmd.AddCommand(agents.NewInstallRunnerCommand())
	cmd.AddCommand(agents.NewInstallCRDCommand())

	return cmd
}
