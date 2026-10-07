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
			printBanner()
			tracker := newInstallTracker()
			if tracker.Enabled() {
				ui.Printf("%s\n", ui.LightGray(telemetry.InstallNotice))
			}
			tracker.Send("install_local_started", nil)
			printPlan()

			printStep(1, "License")
			key := runLicenseStep(tracker, licenseKey)
			// Opted-out users' keys must not reach the owner lookup.
			if tracker.Enabled() {
				tracker.Identify(telemetry.GetEmail(key))
			}

			// Finds tools a previous run installed.
			_ = localinstall.AddToolsDirToPath()
			printStep(2, "Checks")
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
			installMissingTools(cmd, tracker, results)
			tracker.Wait()
		},
	}
	cmd.Flags().StringVarP(&licenseKey, "license", "l", "", "Testkube license key from your trial email (or set TESTKUBE_LICENSE)")
	return cmd
}

func runLicenseStep(tracker *telemetry.InstallTracker, flagKey string) string {
	// curl | bash can't pass flags easily; a variable can.
	if flagKey == "" {
		flagKey = os.Getenv("TESTKUBE_LICENSE")
	}
	if flagKey == "" && !ui.StdinIsInteractive() {
		failLicense(tracker, localinstall.Result{Name: "license", Status: localinstall.StatusFail, Detail: "no key given",
			Fix: "Pass it with --license <key> or TESTKUBE_LICENSE=<key>. " + localinstall.LicenseHelp})
	}
	license, attempts, err := localinstall.NewLicenseStep(terminalKeyPrompter{tracker: tracker}).Run(flagKey)
	for _, a := range attempts {
		tracker.Send("install_local_license", map[string]any{"attempt": a.Number, "status": a.Status})
	}
	if err == nil {
		detail := "valid"
		if !license.Expiry.IsZero() {
			detail = "valid until " + license.Expiry.Local().Format("2 Jan 2006")
		}
		printCheckResult(localinstall.Result{Name: "license", Status: localinstall.StatusPass, Detail: detail})
		return license.Key
	}
	failLicense(tracker, licenseFailure(err))
	return "" // failLicense exits
}

func licenseFailure(err error) localinstall.Result {
	r := localinstall.Result{Name: "license", Status: localinstall.StatusFail, Fix: localinstall.LicenseHelp}
	switch {
	case errors.Is(err, localinstall.ErrLicenseUnreachable):
		r.Detail, r.Fix = "cannot reach license.testkube.io", localinstall.LicenseUnreachableHelp
	case errors.Is(err, localinstall.ErrLicenseMissing):
		r.Detail = "no key entered"
	case errors.Is(err, localinstall.ErrLicenseInvalid):
		r.Detail = "not valid"
	default:
		r.Detail, r.Fix = "could not read the key", "Pass it with --license <key>"
	}
	return r
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

func printBanner() {
	pterm.DefaultBox.WithBoxStyle(pterm.NewStyle(pterm.FgLightMagenta)).Println(
		pterm.Bold.Sprint("Testkube On-Prem Installer") + "  " + ui.LightGray(common.Version) +
			"\nTry Testkube on your own machine.")
}

// Numbered like the step headers, so [3/5] means line 3.
func printPlan() {
	ui.Printf("\nThis installer will\n" +
		"  1. check your license key\n" +
		"  2. check Docker and this machine\n" +
		"  3. install kubectl, helm and kind into ~/.testkube/bin, if missing\n" +
		"  4. create a local cluster called \"testkube\" in Docker\n" +
		"  5. install Testkube and open it in your browser\n\n")
	ui.Printf("%s\n", ui.LightGray("It takes about 5 minutes. Your own tools and clusters are not changed.\n"+
		"Press Ctrl+C to exit at any time."))
}

// Later slices add Cluster and Testkube.
const installSteps = 5

func printStep(n int, title string) {
	ui.Printf("\n%s %s\n", ui.LightGray(fmt.Sprintf("[%d/%d]", n, installSteps)), title)
}

func installMissingTools(cmd *cobra.Command, tracker *telemetry.InstallTracker, results []localinstall.Result) {
	printStep(3, "Tools")
	var missing []string
	for _, r := range results {
		if r.Status == localinstall.StatusWarn && localinstall.ToolVersion(r.Name) != "" {
			missing = append(missing, r.Name)
		}
	}
	if len(missing) == 0 {
		printCheckResult(localinstall.Result{Name: "tools", Status: localinstall.StatusPass, Detail: "nothing to install"})
		return
	}
	installer, err := localinstall.NewToolInstaller()
	if err != nil {
		failToolInstall(tracker, missing[0], err)
	}
	for _, name := range missing {
		version := localinstall.ToolVersion(name)
		spinner := ui.NewSpinner(fmt.Sprintf("Installing %s %s", name, version))
		_, err := installer.Install(cmd.Context(), name)
		// Its own success line would clash with our check rows.
		spinner.RemoveWhenDone = true
		_ = spinner.Stop()
		exitIfCancelled(cmd, tracker, "tool_install")
		if err != nil {
			failToolInstall(tracker, name, err)
		}
		printCheckResult(localinstall.Result{Name: name, Status: localinstall.StatusPass, Detail: version + " installed"})
	}
	tracker.Send("install_local_tools_installed", map[string]any{"tools": missing})
}

func failToolInstall(tracker *telemetry.InstallTracker, name string, err error) {
	r := localinstall.Result{Name: name, Status: localinstall.StatusFail}
	// Fixed reasons only: raw errors can leak paths or proxies.
	var reason string
	switch {
	case errors.Is(err, localinstall.ErrUnsupportedPlatform):
		reason, r.Detail = "unsupported_platform", "no download for this system"
		r.Fix = "Install " + name + " yourself, then run again: " + localinstall.ToolManualURL(name)
	case errors.Is(err, localinstall.ErrChecksumMismatch):
		reason, r.Detail = "checksum", "download was damaged"
		r.Fix = "Run again. If it keeps failing, a proxy may be changing downloads"
	case errors.Is(err, localinstall.ErrSaveFailed):
		reason, r.Detail = "save", "could not save"
		r.Fix = err.Error() + "\nCheck that you can write to ~/.testkube/bin, then run again"
	default:
		reason, r.Detail = "download", "could not download"
		r.Fix = err.Error() + "\nCheck your network, proxy or firewall, then run again"
	}
	printCheckResult(r)
	tracker.Send("install_local_failed", map[string]any{"stage": "tool_install", "tool": name, "reason": reason})
	tracker.Wait()
	os.Exit(1)
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
