package scripts

import (
	"github.com/spf13/cobra"
)

var StartScriptCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts new script",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		println("Starting script")
	},
}
