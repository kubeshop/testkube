package commands

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/scripts"
	"github.com/kubeshop/testkube/pkg/ui"
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
	RootCmd.AddCommand(NewDashboardCmd())
}

var RootCmd = &cobra.Command{
	Use:   "testkube",
	Short: "testkube entrypoint for plugin",
	Long:  `testkube`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		cmd.Usage()
		cmd.DisableAutoGenTag = true
	},

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ui.Verbose = verbose

		// version validation
		// if client version is less than server version show warning
		client, _ := scripts.GetClient(cmd)
		info, err := client.GetServerInfo()
		ui.ExitOnError("getting server info in namespace"+namespace, err)

		serverVersion, err := semver.NewVersion(info.Version)
		if err != nil {
			ui.PrintOnError("parsing server version: "+info.Version, err)
			return
		}

		clientVersion, err := semver.NewVersion(Version)
		if err != nil {
			ui.PrintOnError("parsing client version: "+Version, err)
			return
		}

		if clientVersion.LessThan(serverVersion) {
			ui.Warn("Your TestKube API version is newer than your `kubectl testkube` plugin")
			ui.Info("Testkube API version", serverVersion.String())
			ui.Info("Testkube kubectl plugin client", clientVersion.String())
			ui.Info("It's recommended to upgrade client to version close to API server version")
		}
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
