package scripts

import (
	"fmt"
	"time"

	"github.com/kubeshop/kubetest/pkg/api/client"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

const WatchInterval = 2 * time.Second

var StartScriptCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts new script",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		if len(args) == 0 {
			ui.ExitOnError("Invalid arguments", fmt.Errorf("please pass script name to run"))
		}
		id := args[0]

		client := client.NewRESTClient(client.DefaultURI)
		scriptExecution, err := client.Execute(id)
		ui.ExitOnError("starting script execution", err)

		scriptExecution, err = client.GetExecution(id, scriptExecution.Id)
		ui.ExitOnError("watching API for script completion", err)
		if scriptExecution.Execution.IsCompleted() {
			ui.Success("script completed with sucess")
			// TODO some renderer should be used here based on outpu type
			ui.Info("ID", scriptExecution.Id)
			ui.Info("Output")
			fmt.Println(scriptExecution.Execution.Output)

			ui.ShellCommand(
				"Use following command to get script execution details",
				"kubectl kubetest scripts execution test "+scriptExecution.Id,
			)
			return
		}
	},
}
