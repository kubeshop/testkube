package scripts

import (
	"fmt"

	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func NewAbortExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abort",
		Short: "Aborts execution of the script",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			var executionID string
			if len(args) == 0 {
				ui.Failf("Please pass execution ID as argument")
			}

			executionID = args[0]

			client, _ := GetClient(cmd)

			err := client.AbortExecution("script", executionID)
			ui.ExitOnError(fmt.Sprintf("aborting execution %s", executionID), err)
		},
	}
}
