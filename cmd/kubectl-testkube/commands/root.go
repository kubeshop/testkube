package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	client       string
	verbose      bool
	namespace    string
	oauthEnabled bool
)

func init() {
	// New commands
	RootCmd.AddCommand(NewCreateCmd())
	RootCmd.AddCommand(NewUpdateCmd())

	RootCmd.AddCommand(NewGetCmd())
	RootCmd.AddCommand(NewSetCmd())
	RootCmd.AddCommand(NewRunCmd())
	RootCmd.AddCommand(NewDeleteCmd())
	RootCmd.AddCommand(NewAbortCmd())

	RootCmd.AddCommand(NewEnableCmd())
	RootCmd.AddCommand(NewDisableCmd())
	RootCmd.AddCommand(NewStatusCmd())

	RootCmd.AddCommand(NewDownloadCmd())
	RootCmd.AddCommand(NewGenerateCmd())

	RootCmd.AddCommand(NewInitCmd())
	RootCmd.AddCommand(NewUpgradeCmd())
	RootCmd.AddCommand(NewPurgeCmd())
	RootCmd.AddCommand(NewWatchCmd())
	RootCmd.AddCommand(NewDashboardCmd())
	RootCmd.AddCommand(NewMigrateCmd())
	RootCmd.AddCommand(NewVersionCmd())

	RootCmd.AddCommand(NewConfigCmd())
	RootCmd.AddCommand(NewDebugCmd())
	RootCmd.AddCommand(NewCreateTicketCmd())
	RootCmd.SetHelpCommand(NewHelpCmd())
}

var RootCmd = &cobra.Command{
	Use:   "testkube",
	Short: "Testkube entrypoint for kubectl plugin",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ui.SetVerbose(verbose)
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.Load()

		if cfg.TelemetryEnabled {
			ui.Debug("collecting anonymous telemetry data, you can disable it by calling `kubectl testkube disable telemetry`")
			out, err := telemetry.SendCmdEvent(cmd, common.Version)
			if ui.Verbose && err != nil {
				ui.Err(err)
			}
			ui.Debug("telemetry send event response", out)

			// trigger init event only for first run
			cfg, err := config.Load()
			ui.WarnOnError("loading config", err)

			if !cfg.Initialized {
				cfg.SetInitialized()
				err := config.Save(cfg)
				ui.WarnOnError("saving config", err)

				ui.Debug("sending 'init' event")

				out, err := telemetry.SendCmdInitEvent(cmd, common.Version)
				if ui.Verbose && err != nil {
					ui.Err(err)
				}
				ui.Debug("telemetry init event response", out)
			}
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		err := cmd.Usage()
		ui.PrintOnError("Displaying usage", err)
		cmd.DisableAutoGenTag = true
	},
}

func Execute() {
	cfg, err := config.Load()
	ui.WarnOnError("loading config", err)

	defaultNamespace := "testkube"
	if cfg.Namespace != "" {
		defaultNamespace = cfg.Namespace
	}

	apiURI := "http://localhost:8088"
	if cfg.APIURI != "" {
		apiURI = cfg.APIURI
	}

	if os.Getenv("TESTKUBE_API_URI") != "" {
		apiURI = os.Getenv("TESTKUBE_API_URI")
	}

	RootCmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "client used for connecting to Testkube API one of proxy|direct")
	RootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "", defaultNamespace, "Kubernetes namespace, default value read from config if set")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "show additional debug messages")
	RootCmd.PersistentFlags().StringVarP(&apiURI, "api-uri", "a", apiURI, "api uri, default value read from config if set")
	RootCmd.PersistentFlags().BoolVarP(&oauthEnabled, "oauth-enabled", "", cfg.OAuth2Data.Enabled, "enable oauth")
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
