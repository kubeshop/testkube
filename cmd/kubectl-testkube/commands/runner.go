package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/pro"
)

func NewRunnerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "runner <command>",
		Aliases: []string{""},
		Short:   "Testkube Runner related commands",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(pro.NewInitCmd())

	return cmd
}
