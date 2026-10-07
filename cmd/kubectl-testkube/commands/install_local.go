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

			printStep(1, "Checking your license")
			key := runLicenseStep(tracker, licenseKey)
			// Opted-out users' keys must not reach the owner lookup.
			if tracker.Enabled() {
				tracker.Identify(telemetry.GetEmail(key))
			}

			// Finds tools a previous run installed.
			_ = localinstall.AddToolsDirToPath()
			printStep(2, "Checking required tools")
			checker := localinstall.NewChecker()
			results := checker.CheckTools(cmd.Context())
			exitIfCancelled(cmd, tracker, "tools")
			for _, r := range results {
				if r.Status == localinstall.StatusWarn {
					r.Hint = "will be installed in step 4"
				}
				printCheckResult(r)
			}
			if localinstall.HasFailure(results) {
				trackChecks(tracker, results, checker.Facts())
				tracker.Send("install_local_failed", map[string]any{"stage": "tools"})
				tracker.Wait()
				os.Exit(1)
			}
			printStep(3, "Checking this machine")
			// Not tracked: "os" would overwrite the event's os property.
			printCheckResult(checker.CheckOS())
			machine := append(checker.CheckMachine(cmd.Context()), checker.CheckNetwork(cmd.Context()))
			exitIfCancelled(cmd, tracker, "machine")
			for _, r := range machine {
				printCheckResult(r)
			}
			trackChecks(tracker, append(results, machine...), checker.Facts())
			tracker.Send("install_local_checks_done", nil)
			installMissingTools(cmd, tracker, checker, results)
			runClusterStep(cmd, tracker)
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

// Numbered like the step headers, so [3/6] means line 3.
func printPlan() {
	ui.Printf("\nThis installer will\n" +
		"  1. check your license key\n" +
		"  2. check the tools it needs\n" +
		"  3. check this machine\n" +
		"  4. download helm and kind into ~/.testkube/bin (kubectl too, if missing)\n" +
		"  5. create a local cluster called \"testkube\" in Docker\n" +
		"  6. install Testkube and open it in your browser\n\n")
	ui.Printf("%s\n", ui.LightGray("It takes about 5 minutes. Your own tools and clusters are not changed.\n"+
		"Press Ctrl+C to exit at any time."))
}

// Later slices add Cluster and Testkube.
const installSteps = 6

func printStep(n int, title string) {
	ui.Printf("\n%s %s\n", ui.LightGray(fmt.Sprintf("[%d/%d]", n, installSteps)), title)
}

func installMissingTools(cmd *cobra.Command, tracker *telemetry.InstallTracker, checker *localinstall.Checker, results []localinstall.Result) {
	printStep(4, "Preparing Testkube's tools")
	ui.Printf("  %s\n", ui.LightGray("Testkube uses its own copies in ~/.testkube/bin. Yours are not changed."))
	var missing []string
	for _, r := range results {
		if r.Status == localinstall.StatusWarn && localinstall.ToolVersion(r.Name) != "" {
			missing = append(missing, r.Name)
		}
	}
	for _, name := range localinstall.OwnTools {
		if checker.NeedsOwn(cmd.Context(), name) {
			missing = append(missing, name)
			continue
		}
		printCheckResult(localinstall.Result{Name: name, Status: localinstall.StatusPass, Version: localinstall.ToolVersion(name), Detail: "ready"})
	}
	if len(missing) == 0 {
		return
	}
	installer, err := localinstall.NewToolInstaller()
	if err != nil {
		failToolInstall(tracker, missing[0], err)
	}
	for _, name := range missing {
		version := localinstall.ToolVersion(name)
		spinner := startSpinner(fmt.Sprintf("Installing %s %s", name, version))
		_, err := installer.Install(cmd.Context(), name)
		_ = spinner.Stop()
		exitIfCancelled(cmd, tracker, "tool_install")
		if err != nil {
			failToolInstall(tracker, name, err)
		}
		printCheckResult(localinstall.Result{Name: name, Status: localinstall.StatusPass, Version: version, Detail: "downloaded, checksum verified"})
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

func runClusterStep(cmd *cobra.Command, tracker *telemetry.InstallTracker) {
	printStep(5, "Creating the cluster")
	cluster, err := localinstall.NewCluster()
	if err != nil {
		failCluster(tracker, "", err)
	}
	spinner := startSpinner(fmt.Sprintf("Starting cluster %q (about 1 minute on first run)", localinstall.ClusterName))
	created, out, err := cluster.Ensure(cmd.Context())
	_ = spinner.Stop()
	exitIfCancelled(cmd, tracker, "cluster")
	if err != nil {
		failCluster(tracker, out, err)
	}
	detail := "already exists"
	if created {
		detail = "created"
	}
	printCheckResult(localinstall.Result{Name: "cluster", Status: localinstall.StatusPass, Version: localinstall.KubernetesVersion, Detail: detail})
	tracker.Send("install_local_cluster", map[string]any{"created": created})
}

// pterm's light white text vanishes on light terminals.
// Removed when done: its success line would clash with our rows.
func startSpinner(text string) *pterm.SpinnerPrinter {
	spinner, _ := pterm.DefaultSpinner.
		WithSequence(` ⠋ `, ` ⠹ `, ` ⠼ `, ` ⠦ `, ` ⠇ `).
		WithMessageStyle(pterm.NewStyle(pterm.FgDefault)).
		WithRemoveWhenDone(true).
		Start(text)
	return spinner
}

func failCluster(tracker *telemetry.InstallTracker, out string, err error) {
	reason := lastLines(out, 5)
	if reason == "" {
		reason = err.Error()
	}
	printCheckResult(localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not create",
		Fix: reason + "\nCheck that Docker has enough memory and disk, then run again"})
	tracker.Send("install_local_failed", map[string]any{"stage": "cluster"})
	tracker.Wait()
	os.Exit(1)
}

// kind prints progress first; the reason is at the end.
func lastLines(out string, n int) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
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
	// Padded before coloring: escape codes break %-12s widths.
	line := r.Detail
	switch {
	case r.Version != "":
		line = ui.LightGray(fmt.Sprintf("%-12s", r.Version)) + r.Detail
	case r.Hint != "":
		line = fmt.Sprintf("%-12s", r.Detail) + ui.LightGray(r.Hint)
	}
	ui.Printf("  %s %-10s %s\n", icon, r.Name, line)
	if r.Fix == "" {
		return
	}
	for _, line := range strings.Split(r.Fix, "\n") {
		ui.Printf("      %s\n", line)
	}
}
