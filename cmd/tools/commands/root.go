package commands

import (
	"fmt"
	"os"

	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	Commit  string
	Version string
	BuiltBy string
	Date    string
)

func init() {
	RootCmd.AddCommand(NewReleaseCmd())
}

var RootCmd = &cobra.Command{
	Use:   "",
	Short: "tools command",
	Long:  `tools command`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		cmd.Usage()
		cmd.DisableAutoGenTag = true
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
