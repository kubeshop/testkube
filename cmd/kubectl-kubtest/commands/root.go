package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Commit  string
	Version string
	BuiltBy string
	Date    string
)

var RootCmd = &cobra.Command{
	Use:   "",
	Short: "kubtest entrypoint for plugin",
	Long:  `kubtest`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
