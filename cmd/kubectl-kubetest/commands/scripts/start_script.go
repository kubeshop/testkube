package scripts

import (
	"fmt"
	"time"

	"github.com/kubeshop/kubetest/pkg/api/client"
	"github.com/spf13/cobra"
)

const WatchInterval = 2 * time.Second

var StartScriptCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts new script",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			panic("Please pass script name to run")
		}
		id := args[0]

		client := client.NewRESTClient(client.DefaultURI)
		execution, err := client.Execute(id)
		if err != nil {
			panic(err) // TODO add UI lib for cli apps
		}

		ticker := time.NewTicker(WatchInterval)
		for range ticker.C {
			execution, err = client.GetExecution(id, execution.Id)
			if err != nil {
				panic(err)
			}
			if execution.IsCompleted() {
				// TODO some renderer should be used here based on outpu type
				fmt.Printf("ID:%s\noutput:\n%s", execution.Id, execution.Output)
				return
			}
		}
	},
}
