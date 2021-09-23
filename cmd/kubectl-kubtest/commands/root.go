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
	RootCmd.AddCommand(NewDocsCmd())
	RootCmd.AddCommand(NewScriptsCmd())
	RootCmd.AddCommand(NewVersionCmd())
	RootCmd.AddCommand(NewInstallCmd())
	RootCmd.AddCommand(NewUninstallCmd())
}

var RootCmd = &cobra.Command{
	Use:   "kubtest",
	Short: "kubtest entrypoint for plugin",
	Long:  `kubtest`,
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
