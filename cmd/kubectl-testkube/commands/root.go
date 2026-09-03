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
	localcmd "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/local"
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
	skipTLS   bool
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
	RootCmd.AddCommand(NewViewCmd())
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
	RootCmd.AddCommand(NewMarketplaceCmd())
	RootCmd.AddCommand(localcmd.NewLocalCmd())
	RootCmd.AddCommand(pro.NewLoginCmd())
	RootCmd.AddCommand(NewInstallCmd())

	RootCmd.AddCommand(devbox.NewDevBoxCommand())

	RootCmd.SetHelpCommand(NewHelpCmd())
	RootCmd.AddCommand(NewCompletionCmd())
}

var RootCmd = &cobra.Command{
	Use:   "testkube",
	Short: "Testkube entrypoint for kubectl plugin",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ui.SetVerbose(verbose)

		// Local commands operate directly against a developer-selected Kubernetes
		// cluster and must not depend on, validate, or report to a Testkube API.
		// Check the full ancestry because Cobra invokes this hook with the leaf
		// command (for example, "run" for "testkube local run").
		if isLocalCommand(cmd) {
			return
		}

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
		// Keep local commands fully offline after execution too: the ordinary
		// post-run lifecycle creates an API client, syncs telemetry settings,
		// emits telemetry, and performs an update check.
		if isLocalCommand(cmd) {
			return
		}

		clientCfg, err := config.Load()
		ui.WarnOnError("loading config", err)

		client, _, err := common.GetClient(cmd)
		if err != nil {
			return
		}

		// We ignore this check for cloud, since agent can be offline, and config API won't work
		// but other commands should work.
		cfgDirty := false
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
				cfgDirty = true
			}
		}

		if !isPreRunTelemetry(cmd) {
			handleTelemetry(cmd, &clientCfg, false)
		}

		// Update-check hint - silent in CI/Docker/Kubernetes, skipped for the
		// version command (which renders its own richer status block).
		if common.MaybeNotifyNewerRelease(cmd, &clientCfg) {
			cfgDirty = true
		}

		if cfgDirty {
			if err := config.Save(clientCfg); err != nil {
				ui.WarnOnError("syncing config", err)
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

// isLocalCommand reports whether cmd is in the local command family. Cobra
// passes the leaf command to persistent hooks, so checking only cmd.Name()
// would miss invocations such as "testkube local run" and "testkube local
// clean".
func isLocalCommand(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Name() == "local" {
			return true
		}
	}
	return false
}

// exitCoder is intentionally structural so an independently implemented
// command package can return a typed error without importing this parent
// package (which would create an import cycle). The root process preserves
// the requested non-zero code; all existing errors retain the historical
// exit status of 1.
type exitCoder interface {
	error
	ExitCode() int
}

func exitCodeForError(err error) int {
	var coded exitCoder
	if errors.As(err, &coded) && coded.ExitCode() > 0 {
		return coded.ExitCode()
	}
	return 1
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
		ui.Debug("collecting anonymous telemetry data, you can disable it by calling `testkube disable telemetry`")
		userID := common.TelemetryUserID(cmd, cfg)
		out, err := telemetry.SendCmdEvent(cmd, common.Version, userID)
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

			out, err := telemetry.SendCmdInitEvent(cmd, common.Version, userID)
			if ui.Verbose && err != nil {
				ui.Err(err)
			}
			ui.Debug("telemetry init event response", out)
		}
	}
}

func handleCLIErrorTelemetry(version string, err *common.CLIError) (string, error) {
	if err.Telemetry == nil {
		return "", nil
	}
	return telemetry.SendCmdErrorEventWithLicense(
		err.Telemetry.Command,
		version,
		err.Telemetry.Type,
		err.StackTrace,
		err.Telemetry.License,
		err.Telemetry.Step,
		string(err.Code),
	)
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
	RootCmd.PersistentFlags().BoolVarP(&skipTLS, "skip-tls", "", false, "skip TLS certificate verification for backend HTTPS connections")
	RootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "", false, "deprecated: use --skip-tls")
	RootCmd.PersistentFlags().MarkDeprecated("insecure", "use --skip-tls")
	RootCmd.PersistentFlags().StringToStringVarP(&headers, "header", "", cfg.Headers, "headers for direct client key value pair: --header name=value")

	if err := RootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeForError(err))
	}
}
