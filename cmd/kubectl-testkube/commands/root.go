package commands

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/scripts"
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
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
	RootCmd.AddCommand(NewExecutorsCmd())
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

		err := ValidateVersions(client)
		if err != nil {
			ui.Warn(err.Error())
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
		ui.Warn("Your TestKube API version is newer than your `kubectl testkube` plugin")
		ui.Info("Testkube API version", serverVersion.String())
		ui.Info("Testkube kubectl plugin client", clientVersion.String())
		ui.Info("It's recommended to upgrade client to version close to API server version")
	}

	return nil
}

func Execute() {
	RootCmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	RootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "testkube", "kubernetes namespace")

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
