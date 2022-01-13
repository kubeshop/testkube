package executors

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetExecutorCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Gets executordetails",
		Long:  `Gets executor, you can change output format`,
		Run: func(cmd *cobra.Command, args []string) {

			if len(args) == 1 {
				name = args[0]
			}

			if name == "" {
				ui.Failf("Please pass executor name")
			}

			client, _ := GetClient(cmd)
			executor, err := client.GetExecutor(name)
			ui.ExitOnError("getting executor: "+name, err)

			render := GetExecutorRenderer(cmd)
			err = render.Render(executor, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique executor name, you can also pass it as argument")

	return cmd
}
