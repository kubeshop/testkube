package commands

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInstallLocalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "local",
		Short: "Install Testkube on this machine",
		// Hidden until the guided installer is complete.
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			tracker := newInstallTracker()
			if tracker.Enabled() {
				ui.Printf("%s\n\n", ui.LightGray(localinstall.Notice))
			}
			tracker.Send("install_local_started", nil)

			checker := localinstall.NewChecker()
			results := checker.CheckTools(cmd.Context())
			// Ctrl+C is the user quitting, not Docker failing.
			if cmd.Context().Err() != nil {
				os.Exit(130)
			}
			for _, r := range results {
				printCheckResult(r)
				trackCheck(tracker, r)
			}
			if localinstall.HasFailure(results) {
				tracker.Send("install_local_failed", map[string]any{"stage": "tools"})
				tracker.Wait()
				os.Exit(1)
			}
			for _, r := range checker.CheckMachine(cmd.Context()) {
				printCheckResult(r)
				trackCheck(tracker, r)
			}
			tracker.Send("install_local_checks_done", nil)
			tracker.Wait()
		},
	}
}

func newInstallTracker() *localinstall.Tracker {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig
	}
	return localinstall.NewTracker(cfg.TelemetryEnabled, telemetry.GetMachineID(), common.Version)
}

func trackCheck(tracker *localinstall.Tracker, r localinstall.Result) {
	tracker.Send("install_local_check", map[string]any{"check": r.Name, "status": string(r.Status)})
}

func printCheckResult(r localinstall.Result) {
	icon := map[localinstall.Status]string{
		localinstall.StatusPass: ui.Green("✔"),
		localinstall.StatusWarn: ui.LightYellow("⚠"),
		localinstall.StatusFail: ui.LightRed("✖"),
	}[r.Status]
	ui.Printf("  %s %-8s %s\n", icon, r.Name, ui.LightGray(r.Detail))
	if r.Fix == "" {
		return
	}
	for _, line := range strings.Split(r.Fix, "\n") {
		ui.Printf("      %s\n", line)
	}
}
