package scripts

import (
	"os"

	"github.com/kubeshop/kubetest/pkg/api/client"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

var GetScriptExecutionsCmd = &cobra.Command{
	Use:   "list-executions",
	Short: "Gets script executions details",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			ui.Errf("invlid script arguments please pass test name and execution id")
		}

		scriptID := args[0]

		client := client.NewRESTClient(client.DefaultURI)
		executions, err := client.GetExecutions(scriptID)
		ui.ExitOnError("getting executions ", err)

		ui.Table(executions, os.Stdout)

	},
}
