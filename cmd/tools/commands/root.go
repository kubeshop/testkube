package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	Commit  string
	Version string
	BuiltBy string
	Date    string
)

func init() {
	RootCmd.AddCommand(NewReleaseCmd())
	RootCmd.AddCommand(NewVersionBumpCmd())
}

var RootCmd = &cobra.Command{
	Use:   "",
	Short: "tools command",
	Long:  `tools command`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		err := cmd.Usage()
		ui.PrintOnError("Displaying usage", err)
		cmd.DisableAutoGenTag = true
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
