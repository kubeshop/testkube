package scripts

import (
	"github.com/spf13/cobra"
)

var GetScriptExecutionCmd = &cobra.Command{
	Use:   "execution",
	Short: "Gets script execution details",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		println("Script exection details")
	},
}
