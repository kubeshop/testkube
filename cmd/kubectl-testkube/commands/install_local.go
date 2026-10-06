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
				ui.Printf("%s\n\n", ui.LightGray(telemetry.InstallNotice))
			}
			tracker.Send("install_local_started", nil)

			checker := localinstall.NewChecker()
			results := checker.CheckTools(cmd.Context())
			exitIfCancelled(cmd, tracker, "tools")
			for _, r := range results {
				printCheckResult(r)
			}
			if localinstall.HasFailure(results) {
				trackChecks(tracker, results, checker.Facts())
				tracker.Send("install_local_failed", map[string]any{"stage": "tools"})
				tracker.Wait()
				os.Exit(1)
			}
			machine := checker.CheckMachine(cmd.Context())
			exitIfCancelled(cmd, tracker, "machine")
			for _, r := range machine {
				printCheckResult(r)
			}
			trackChecks(tracker, append(results, machine...), checker.Facts())
			tracker.Send("install_local_checks_done", nil)
			tracker.Wait()
		},
	}
}

func newInstallTracker() *telemetry.InstallTracker {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig
	}
	return telemetry.NewInstallTracker(telemetry.InstallTrackerConfig{
		Enabled:   cfg.TelemetryEnabled && !telemetry.DoNotTrack(),
		MachineID: telemetry.GetMachineID(),
		Version:   common.Version,
	})
}

// Ctrl+C is the user quitting, not a step failing.
func exitIfCancelled(cmd *cobra.Command, tracker *telemetry.InstallTracker, stage string) {
	if cmd.Context().Err() == nil {
		return
	}
	tracker.Send("install_local_aborted", map[string]any{"stage": stage})
	tracker.Wait()
	os.Exit(130)
}

// One row per run: every status and measured value together.
func trackChecks(tracker *telemetry.InstallTracker, results []localinstall.Result, facts map[string]any) {
	props := map[string]any{}
	for k, v := range facts {
		props[k] = v
	}
	for _, r := range results {
		props[r.Name] = string(r.Status)
	}
	tracker.Send("install_local_checks", props)
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
