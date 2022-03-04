package commands

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/analytics"
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
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

	RootCmd.AddCommand(NewTestsCmd())
	RootCmd.AddCommand(NewVersionCmd())
	RootCmd.AddCommand(NewInstallCmd())
	RootCmd.AddCommand(NewUpgradeCmd())
	RootCmd.AddCommand(NewUninstallCmd())
	RootCmd.AddCommand(NewDashboardCmd())
	RootCmd.AddCommand(NewExecutorsCmd())
	RootCmd.AddCommand(NewTestSuitesCmd())
	RootCmd.AddCommand(NewMigrateCmd())
	RootCmd.AddCommand(NewWebhooksCmd())
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

		if analyticsEnabled {
			ui.Debug("collecting anonymous analytics data, you can disable it by calling `kubectl testkube anlytics disable`")
			analytics.SendAnonymouscmdInfo()
		}
	},
}

func ValidateVersions(c apiclient.Client) error {
	info, err := c.GetServerInfo()
	if err != nil {
		return fmt.Errorf("getting server info: %w", err)
	}

	serverVersion, err := semver.NewVersion(info.Version)
	if err != nil {
		return fmt.Errorf("parsing server version - %s: %w", info.Version, err)
	}

	clientVersion, err := semver.NewVersion(Version)
	if err != nil {
		return fmt.Errorf("parsing client version - %s: %w", Version, err)
	}

	if clientVersion.LessThan(serverVersion) {
		ui.Warn("Your Testkube API version is newer than your `kubectl testkube` plugin")
		ui.Info("Testkube API version", serverVersion.String())
		ui.Info("Testkube kubectl plugin client", clientVersion.String())
		ui.Info("It's recommended to upgrade client to version close to API server version")
		ui.NL()
	}

	return nil
}

func Execute() {
	cfg, err := config.Load()
	ui.WarnOnError("Config loading error", err)

	RootCmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	RootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "testkube", "kubernetes namespace")
	RootCmd.PersistentFlags().BoolVarP(&analyticsEnabled, "analytics-enabled", "", cfg.AnalyticsEnabled, "should analytics be enabled")

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
