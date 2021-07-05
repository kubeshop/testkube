package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "kubetest",
	Short: "Kubetest entrypoint for plugin",
	Long:  `Kubetest`,
	Run: func(cmd *cobra.Command, args []string) {
		println("ROOT")
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
