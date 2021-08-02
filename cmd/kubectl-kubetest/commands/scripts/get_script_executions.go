package scripts

import (
	"os"

	"github.com/kubeshop/kubetest/pkg/api/client"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

var GetScriptExecutionsCmd = &cobra.Command{
	Use:   "executions",
	Short: "Gets script executions list",
	Long:  `Getting list of execution for given script name`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			ui.Failf("invalid script arguments please pass test name")
		}

		scriptID := args[0]
		client := client.NewScriptsAPI(client.DefaultURI)

		executions, err := client.GetExecutions(scriptID)
		ui.ExitOnError("getting executions ", err)
		ui.Table(executions, os.Stdout)
	},
}
