package scripts

import (
	"github.com/spf13/cobra"
)

var AbortExecutionCmd = &cobra.Command{
	Use:   "abort",
	Short: "Aborts execution of the script",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		println("Aborting")
	},
}
