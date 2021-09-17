package scripts

import (
	"fmt"

	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func NewAbortExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abort",
		Short: "(NOT IMPLEMENTED) Aborts execution of the script",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			var scriptID string
			if len(args) == 0 {
				scriptID = "-"
			} else if len(args) > 0 {
				scriptID = args[0]
			}

			client, _ := GetClient(cmd)
			fmt.Println("....")
			err := client.AbortExecution("script", scriptID)
			ui.ExitOnError(fmt.Sprintf("aborting execution %s", scriptID), err)
		},
	}
}
