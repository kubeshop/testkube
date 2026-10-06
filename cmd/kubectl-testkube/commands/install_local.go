package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInstallLocalCmd() *cobra.Command {
	var licenseKey string
	cmd := &cobra.Command{
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

			runLicenseStep(tracker, licenseKey)

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
	cmd.Flags().StringVarP(&licenseKey, "license", "l", "", "Testkube license key from your trial email")
	return cmd
}

func runLicenseStep(tracker *telemetry.InstallTracker, flagKey string) {
	if flagKey == "" && !ui.StdinIsInteractive() {
		failLicense(tracker, localinstall.Result{Name: "license", Status: localinstall.StatusFail, Detail: "no key given",
			Fix: "Pass it with --license <key>. " + localinstall.LicenseHelp})
	}
	_, attempts, err := localinstall.NewLicenseStep(terminalKeyPrompter{tracker: tracker}).Run(flagKey)
	for _, a := range attempts {
		tracker.Send("install_local_license", map[string]any{"attempt": a.Number, "status": a.Status})
	}
	switch {
	case err == nil:
		printCheckResult(localinstall.Result{Name: "license", Status: localinstall.StatusPass, Detail: "valid"})
	case errors.Is(err, localinstall.ErrLicenseUnreachable):
		failLicense(tracker, localinstall.Result{Name: "license", Status: localinstall.StatusFail, Detail: "cannot reach license.testkube.io", Fix: localinstall.LicenseUnreachableHelp})
	case errors.Is(err, localinstall.ErrLicenseMissing):
		failLicense(tracker, localinstall.Result{Name: "license", Status: localinstall.StatusFail, Detail: "no key entered", Fix: localinstall.LicenseHelp})
	case errors.Is(err, localinstall.ErrLicenseInvalid):
		failLicense(tracker, localinstall.Result{Name: "license", Status: localinstall.StatusFail, Detail: "not valid", Fix: localinstall.LicenseHelp})
	default:
		failLicense(tracker, localinstall.Result{Name: "license", Status: localinstall.StatusFail, Detail: "could not read the key", Fix: "Pass it with --license <key>"})
	}
}

func failLicense(tracker *telemetry.InstallTracker, r localinstall.Result) {
	printCheckResult(r)
	tracker.Send("install_local_failed", map[string]any{"stage": "license"})
	tracker.Wait()
	os.Exit(1)
}

type terminalKeyPrompter struct {
	tracker *telemetry.InstallTracker
}

func (p terminalKeyPrompter) Ask(attempt int) (string, error) {
	label := "Enter your Testkube license key"
	if attempt > 1 {
		label = fmt.Sprintf("That key is not valid. Try again (%d of %d)", attempt, localinstall.MaxLicenseAttempts)
	}
	return pterm.DefaultInteractiveTextInput.
		WithMask("*").
		WithOnInterruptFunc(func() { abortInstall(p.tracker, "license") }).
		Show(label)
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
	abortInstall(tracker, stage)
}

func abortInstall(tracker *telemetry.InstallTracker, stage string) {
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
