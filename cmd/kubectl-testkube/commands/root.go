package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/analytics"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	Commit  string
	Version string
	BuiltBy string
	Date    string

	analyticsEnabled bool
	client           string
	verbose          bool
	namespace        string
)

func init() {

	// New commands
	RootCmd.AddCommand(NewCreateCmd())
	RootCmd.AddCommand(NewGetCmd())
	RootCmd.AddCommand(NewRunCmd())
	RootCmd.AddCommand(NewDeleteCmd())
	RootCmd.AddCommand(NewAbortCmd())

	RootCmd.AddCommand(NewEnableCmd())
	RootCmd.AddCommand(NewDisableCmd())
	RootCmd.AddCommand(NewStatusCmd())

	RootCmd.AddCommand(NewDownloadCmd())
	RootCmd.AddCommand(NewGenerateCmd())

	RootCmd.AddCommand(NewInstallCmd())
	RootCmd.AddCommand(NewUpgradeCmd())
	RootCmd.AddCommand(NewUninstallCmd())
	RootCmd.AddCommand(NewDashboardCmd())
	RootCmd.AddCommand(NewMigrateCmd())
	RootCmd.AddCommand(NewVersionCmd())
}

var RootCmd = &cobra.Command{
	Use:   "testkube",
	Short: "Testkube entrypoint for kubectl plugin",
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		cmd.Usage()
		cmd.DisableAutoGenTag = true
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		ui.Verbose = verbose

		if analyticsEnabled {
			ui.Debug("collecting anonymous analytics data, you can disable it by calling `kubectl testkube disable analytics`")
			analytics.SendAnonymouscmdInfo()
		}
	},
}

func Execute() {
	cfg, err := config.Load()
	ui.WarnOnError("Config loading error", err)

	RootCmd.PersistentFlags().BoolVarP(&analyticsEnabled, "analytics-enabled", "", cfg.AnalyticsEnabled, "enable analytics")
	RootCmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "client used for connecting to Testkube API one of proxy|direct")
	RootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "testkube", "Kubernetes namespace")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show additional debug messages")

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
