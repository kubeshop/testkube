package executors

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteExecutorCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Gets executordetails",
		Long:  `Gets executor, you can change output format`,
		Run: func(cmd *cobra.Command, args []string) {

			client, _ := GetClient(cmd)
			executor, err := client.GetExecutor(name)
			ui.ExitOnError("getting script executor: "+name, err)

			render := GetExecutorRenderer(cmd)
			err = render.Render(executor, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique executor name - mandatory")

	return cmd
}
