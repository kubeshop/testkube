package executors

import (
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListExecutorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Gets executors",
		Long:  `Gets executor, you can change output format`,
		Run: func(cmd *cobra.Command, args []string) {

			client, _ := common.GetClient(cmd)
			executors, err := client.ListExecutors()
			ui.ExitOnError("listing executors: ", err)

			render := GetExecutorListRenderer(cmd)
			err = render.Render(executors, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}

	return cmd
}
