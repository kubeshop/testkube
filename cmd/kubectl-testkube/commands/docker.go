package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/docker"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDockerCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Testkube Docker commands",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
	}

	cmd.AddCommand(docker.NewInitCmd())

	return cmd
}
