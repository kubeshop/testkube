package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/pro"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/cmd/tcl/kubectl-testkube/devbox"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	client    string
	verbose   bool
	namespace string
	insecure  bool
	headers   map[string]string
)

// preRunTelemetryCommands defines which commands should send telemetry in PreRun
// These are typically long-running or blocking commands that won't reach PostRun until completion
var preRunTelemetryCommands = map[string]string{
	"serve": "mcp", // serve command under mcp parent - this won't reach PostRun until the server is stopped
}

func init() {
	// New commands
	RootCmd.AddCommand(NewCreateCmd())
	RootCmd.AddCommand(NewUpdateCmd())

	RootCmd.AddCommand(NewGetCmd())
	RootCmd.AddCommand(NewSetCmd())
	RootCmd.AddCommand(NewRunCmd())
	RootCmd.AddCommand(NewDeleteCmd())
	RootCmd.AddCommand(NewAbortCmd())
	RootCmd.AddCommand(NewCancelCmd())

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
	RootCmd.AddCommand(NewDiagnosticsCmd())
	RootCmd.AddCommand(NewCreateTicketCmd())

	RootCmd.AddCommand(NewAgentCmd())
	RootCmd.AddCommand(NewCloudCmd())
	RootCmd.AddCommand(NewProCmd())
	RootCmd.AddCommand(NewMcpCmd())
	RootCmd.AddCommand(NewDockerCmd())
	RootCmd.AddCommand(pro.NewLoginCmd())
	RootCmd.AddCommand(NewInstallCmd())

	RootCmd.AddCommand(devbox.NewDevBoxCommand())

	RootCmd.SetHelpCommand(NewHelpCmd())
}

var RootCmd = &cobra.Command{
	Use:   "testkube",
	Short: "Testkube entrypoint for kubectl plugin",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ui.SetVerbose(verbose)

		// don't validate context before set and completion
		if cmd.Name() == "context" || (cmd.Parent() != nil && cmd.Parent().Name() == "completion") {
			return
		}

		cfg, err := config.Load()
		ui.ExitOnError("loading config", err)

		if err = validator.ValidateCloudContext(cfg); err != nil {
			common.UiCloudContextValidationError(err)
		}

		// send telemetry as needed
		if isPreRunTelemetry(cmd) {
			handleTelemetry(cmd, &cfg, true)
		}
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		clientCfg, err := config.Load()
		ui.WarnOnError("loading config", err)

		client, _, err := common.GetClient(cmd)
		if err != nil {
			return
		}

		// We ignore this check for cloud, since agent can be offline, and config API won't work
		// but other commands should work.
		if clientCfg.ContextType != config.ContextTypeCloud {
			serverCfg, err := client.GetConfig()
			if ui.Verbose && err != nil {
				ui.Err(err)
			}

			if clientCfg.TelemetryEnabled != serverCfg.EnableTelemetry && err == nil {
				if serverCfg.EnableTelemetry {
					clientCfg.EnableAnalytics()
					ui.Debug("Sync telemetry on CLI with API", "enabled")
				} else {
					clientCfg.DisableAnalytics()
					ui.Debug("Sync telemetry on CLI with API", "disabled")
				}

				err = config.Save(clientCfg)
				ui.WarnOnError("syncing config", err)
			}
		}

		if !isPreRunTelemetry(cmd) {
			handleTelemetry(cmd, &clientCfg, false)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()
		err := cmd.Usage()
		ui.PrintOnError("Displaying usage", err)
		cmd.DisableAutoGenTag = true
	},
}

// isPreRunTelemetry determines if telemetry should be sent in PreRun for this command
func isPreRunTelemetry(cmd *cobra.Command) bool {
	expectedParent, exists := preRunTelemetryCommands[cmd.Name()]
	if !exists {
		return false
	}
	return cmd.Parent() != nil && cmd.Parent().Name() == expectedParent
}

// handleTelemetry sends telemetry events and handles initialization events
func handleTelemetry(cmd *cobra.Command, cfg *config.Data, isPreRun bool) {
	// Send telemetry early to ensure it's captured even if command fails
	if cfg.TelemetryEnabled {
		ui.Debug("collecting anonymous telemetry data, you can disable it by calling `kubectl testkube disable telemetry`")
		out, err := telemetry.SendCmdEvent(cmd, common.Version)
		if ui.Verbose && err != nil {
			ui.Err(err)
		}
		ui.Debug("telemetry send event response", out)

		// trigger init event only for first run
		if !cfg.Initialized && !isPreRun {
			cfg.SetInitialized()
			err := config.Save(*cfg)
			ui.WarnOnError("saving config", err)

			ui.Debug("sending 'init' event")

			out, err := telemetry.SendCmdInitEvent(cmd, common.Version)
			if ui.Verbose && err != nil {
				ui.Err(err)
			}
			ui.Debug("telemetry init event response", out)
		}
	}
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

	// Run services within an errgroup to propagate errors between services.
	g, ctx := errgroup.WithContext(context.Background())

	// Cancel the errgroup context on SIGINT and SIGTERM,
	// which shuts everything down gracefully.
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-stopSignal:
			go func() {
				<-stopSignal
				os.Exit(137)
			}()
			return errors.Errorf("received signal: %v", sig)
		}
	})

	RootCmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "client used for connecting to Testkube API one of proxy|direct|cluster")
	RootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "", defaultNamespace, "Kubernetes namespace, default value read from config if set")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "show additional debug messages")
	RootCmd.PersistentFlags().StringVarP(&apiURI, "api-uri", "a", apiURI, "api uri, default value read from config if set")
	RootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "", false, "insecure connection for direct client")
	RootCmd.PersistentFlags().StringToStringVarP(&headers, "header", "", cfg.Headers, "headers for direct client key value pair: --header name=value")

	if err := RootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
