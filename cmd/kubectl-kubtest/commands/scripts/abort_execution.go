package scripts

import (
	"github.com/spf13/cobra"
)

func NewAbortExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abort",
		Short: "(NOT IMPLEMENTED) Aborts execution of the script",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			println("Aborting")
		},
	}
}
