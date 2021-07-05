package scripts

import (
	"github.com/spf13/cobra"
)

var ListScriptsCmd = &cobra.Command{
	Use:   "list",
	Short: "Get all available scripts",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		println("Listing all scripts")
	},
}
