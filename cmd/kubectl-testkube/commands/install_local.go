package commands

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	licensevalidator "github.com/kubeshop/testkube/pkg/diagnostics/validators/license"
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
			reportInstallLicenseEvent(tracker, key, licensevalidator.EventCLIInstallStarted)

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
				waitForEvents(tracker)
				os.Exit(1)
			}
			printStep(3, "Checking this machine")
			// Not tracked: "os" would overwrite the event's os property.
			printCheckResult(checker.CheckOS())
			machine := append(checker.CheckMachine(cmd.Context()), checker.CheckInotify(cmd.Context())...)
			machine = append(machine, checker.CheckNetwork(cmd.Context()))
			exitIfCancelled(cmd, tracker, "machine")
			for _, r := range machine {
				printCheckResult(r)
			}
			trackChecks(tracker, append(results, machine...), checker.Facts())
			tracker.Send("install_local_checks_done", nil)
			installMissingTools(cmd, tracker, checker, results)
			ports := runClusterStep(cmd, tracker)
			runTestkubeStep(cmd, tracker, key, ports)
			printReady(tracker, ports)
			reportInstallLicenseEvent(tracker, key, licensevalidator.EventCLIInstallFinished)
			waitForEvents(tracker)
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
	failLicense(tracker, licenseFailure(err, license.Expiry))
	return "" // failLicense exits
}

func licenseFailure(err error, expiry time.Time) localinstall.Result {
	r := localinstall.Result{Name: "license", Status: localinstall.StatusFail, Fix: localinstall.LicenseHelp}
	switch {
	case errors.Is(err, localinstall.ErrLicenseUnreachable):
		r.Detail, r.Fix = "cannot reach license.testkube.io", localinstall.LicenseUnreachableHelp
	case errors.Is(err, localinstall.ErrLicenseMissing):
		r.Detail = "no key entered"
	case errors.Is(err, localinstall.ErrLicenseInvalid):
		r.Detail = "not valid"
	case errors.Is(err, localinstall.ErrLicenseExpired):
		r.Detail, r.Fix = "expired", "This key has expired. Get a new one at https://testkube.io/get-started/on-prem"
		if !expiry.IsZero() {
			r.Detail = "expired on " + expiry.Local().Format("2 Jan 2006")
		}
	default:
		r.Detail, r.Fix = "could not read the key", "Pass it with --license <key>"
	}
	return r
}

func failLicense(tracker *telemetry.InstallTracker, r localinstall.Result) {
	printCheckResult(r)
	tracker.Send("install_local_failed", map[string]any{"stage": "license"})
	waitForEvents(tracker)
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
	waitForEvents(tracker)
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
	ui.Printf("%s\n", ui.LightGray("First run takes 5 to 10 minutes, mostly downloads. Your own tools and clusters are not changed.\n"+
		"Press Ctrl+C to exit at any time."))
}

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
	start := time.Now()
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
	// Measured, so the plan's time hints can match reality.
	tracker.Send("install_local_tools_installed", map[string]any{"tools": missing, "duration_s": int(time.Since(start).Seconds())})
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
	waitForEvents(tracker)
	os.Exit(1)
}

func runClusterStep(cmd *cobra.Command, tracker *telemetry.InstallTracker) localinstall.Ports {
	printStep(5, "Creating the cluster")
	cluster, err := localinstall.NewCluster()
	if err != nil {
		failCluster(tracker, "", err)
	}
	spinner := startSpinner(fmt.Sprintf("Starting cluster %q (about 1 minute on first run)", localinstall.ClusterName))
	start := time.Now()
	state, out, err := cluster.Ensure(cmd.Context())
	_ = spinner.Stop()
	exitIfCancelled(cmd, tracker, "cluster")
	if err != nil {
		failCluster(tracker, out, err)
	}
	detail := "already exists"
	switch {
	case state.Created:
		detail = "created"
	case state.Started:
		detail = "started"
	}
	r := localinstall.Result{Name: "cluster", Status: localinstall.StatusPass, Version: localinstall.KubernetesVersion, Detail: detail}
	if moved := state.Ports.Moved(); len(moved) > 0 {
		r.Hint = "busy port moved: " + strings.Join(moved, ", ")
	}
	printCheckResult(r)
	disk, free := cluster.CheckDisk(cmd.Context())
	printCheckResult(disk)
	ui.Printf("  %s\n", ui.LightGray("Your Testkube data is kept in ~/.testkube/data"))
	props := map[string]any{"created": state.Created, "ports_moved": len(state.Ports.Moved()), "disk": string(disk.Status),
		"duration_s": int(time.Since(start).Seconds())}
	if free > 0 {
		props["node_disk_free_gb"] = free
	}
	tracker.Send("install_local_cluster", props)
	return state.Ports
}

func runTestkubeStep(cmd *cobra.Command, tracker *telemetry.InstallTracker, license string, ports localinstall.Ports) {
	printStep(6, "Installing Testkube")
	secrets, err := localinstall.LoadOrCreateSecrets()
	if err != nil {
		failTestkube(tracker, "", err)
	}
	installer, err := localinstall.NewInstaller()
	if err != nil {
		failTestkube(tracker, "", err)
	}
	start := time.Now()
	waiting := func(elapsed time.Duration) string {
		return "Installing Testkube · " + took(elapsed) + " (first run 3 to 8 minutes)"
	}
	spinner := startSpinner(waiting(0))
	stopTicking := tickElapsed(spinner, start, waiting)
	state, out, err := installer.Install(cmd.Context(), ports, secrets, license)
	stopTicking()
	_ = spinner.Stop()
	exitIfCancelled(cmd, tracker, "testkube")
	if err != nil {
		failTestkube(tracker, out, err)
	}
	r := localinstall.Result{Name: "testkube", Status: localinstall.StatusPass, Version: localinstall.AppVersion,
		Detail: "installed in " + took(state.TestkubeTook)}
	if state.Recovered {
		r.Hint = "cleaned up an interrupted install"
	}
	printCheckResult(r)
	printCheckResult(localinstall.Result{Name: "runner", Status: localinstall.StatusPass, Version: localinstall.AppVersion,
		Detail: "installed in " + took(state.RunnerTook)})
	tracker.Send("install_local_testkube", map[string]any{"recovered": state.Recovered, "migration_retried": state.MigrationRetried,
		"testkube_s": int(state.TestkubeTook.Seconds()), "runner_s": int(state.RunnerTook.Seconds()),
		"duration_s": int(time.Since(start).Seconds())})
}

// Long waits need a visible pulse; download ETAs would lie.
func tickElapsed(spinner *pterm.SpinnerPrinter, start time.Time, text func(time.Duration) string) (stop func()) {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				spinner.UpdateText(text(time.Since(start)))
			}
		}
	}()
	return func() { close(done) }
}

func took(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
}

func failTestkube(tracker *telemetry.InstallTracker, out string, err error) {
	r := localinstall.Result{Name: "testkube", Status: localinstall.StatusFail, Version: localinstall.AppVersion, Detail: "could not install"}
	reason := "install"
	var stuck localinstall.StuckError
	isStuck := errors.As(err, &stuck)
	switch {
	case errors.Is(err, localinstall.ErrSecretsLost):
		reason, r.Detail = "secrets_lost", "old data can't be opened"
		rm := "rm"
		// Linux keeps the database's own user on its files.
		if runtime.GOOS == "linux" {
			rm = "sudo rm"
		}
		r.Fix = "~/.testkube/data holds data from an earlier install whose passwords are gone.\n" +
			"Delete the cluster and that data to start fresh, then run again:\n" +
			"  ~/.testkube/bin/kind delete cluster --name " + localinstall.ClusterName + "\n" +
			"  " + rm + " -rf ~/.testkube/data/" + localinstall.Namespace
	case errors.Is(err, localinstall.ErrSecretsDamaged):
		reason, r.Detail = "secrets_damaged", "saved passwords can't be read"
		r.Fix = "Restore ~/.testkube/secrets.json, or delete it and ~/.testkube/data/" + localinstall.Namespace +
			" to start fresh"
	case errors.Is(err, localinstall.ErrStuck):
		reason, r.Detail = "stuck", "stopped after "+took(stuck.After)
		if errors.Is(err, localinstall.ErrRunnerInstall) {
			r.Name = "runner"
		}
	case errors.Is(err, localinstall.ErrRunnerInstall):
		reason, r.Name = "runner", "runner"
		r.Fix = withWhy(out, "Testkube is installed; run again to retry the runner")
	case errors.Is(err, localinstall.ErrInstallTimeout):
		reason, r.Detail = "timeout", "not ready after 15 minutes"
		r.Fix = withWhy(out, "Run again; finished downloads are kept")
	case errors.Is(err, localinstall.ErrChartDownload):
		reason, r.Detail = "chart_download", "could not download"
		r.Fix = withWhy(out, "Check your network, proxy or firewall, then run again")
	case errors.Is(err, localinstall.ErrTestkubePrepare):
		reason, r.Detail = "prepare", "could not prepare the cluster"
		r.Fix = withWhy(out, "Run again")
	default:
		if out == "" {
			out = err.Error()
		}
		r.Fix = withWhy(out, "Run again")
	}
	if isStuck {
		r.Fix = stuckFix(stuck.Stuck)
	}
	printCheckResult(r)
	tracker.Send("install_local_failed", map[string]any{"stage": "testkube", "reason": reason})
	waitForEvents(tracker)
	os.Exit(1)
}

const contactFix = "Run again. If it fails the same way, tell us at https://testkube.io/contact"

// Plain words for what's stuck; helm's own lines never say why.
func stuckFix(st localinstall.Stuck) string {
	name := st.Service
	switch st.Reason {
	case "downloading":
		return fmt.Sprintf("%s is still downloading %s\nYour connection may be slow. Run again; finished downloads are kept.", name, st.Detail)
	case "image_pull", "rate_limit":
		return fmt.Sprintf("%s can't download %s\nCheck your network, proxy or firewall, then run again. Finished downloads are kept.", name, st.Detail)
	case "image_missing":
		return fmt.Sprintf("%s can't download %s: it doesn't exist or needs a login\n%s", name, st.Detail, contactFix)
	case "no_cpu":
		return fmt.Sprintf("%s can't start: Docker doesn't have enough CPU\nGive Docker 4 CPUs or more, then run again.", name)
	case "no_memory":
		return fmt.Sprintf("%s can't start: Docker doesn't have enough memory\nGive Docker 6 GB or more, then run again.", name)
	case "oom":
		return fmt.Sprintf("%s ran out of memory (limit %s)\n%s", name, st.Detail, contactFix)
	case "storage":
		return fmt.Sprintf("%s can't get its storage: %s\n%s", name, st.Detail, contactFix)
	case "crashloop", "job_failed":
		head := fmt.Sprintf("%s keeps crashing (restarted %d times).", name, st.Restarts)
		if st.Reason == "job_failed" {
			head = name + " failed."
		}
		if len(st.Logs) == 0 {
			return head + "\n" + contactFix
		}
		return head + " Last log lines:\n  " + strings.Join(st.Logs, "\n  ") + "\n" + contactFix
	case "not_ready":
		return fmt.Sprintf("%s is %s\n%s", name, st.Detail, contactFix)
	}
	return fmt.Sprintf("%s can't start: %s\n%s", name, st.Detail, contactFix)
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
	r := localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not create"}
	reason := "create"
	var taken localinstall.PortTakenError
	switch {
	case errors.Is(err, localinstall.ErrClusterNotOurs):
		reason, r.Detail = "not_ours", "name already taken"
		r.Fix = fmt.Sprintf("A kind cluster named %q exists that this installer didn't create.\n"+
			"Delete it if you don't need it, then run again:\n  ~/.testkube/bin/kind delete cluster --name %s", localinstall.ClusterName, localinstall.ClusterName)
	case errors.Is(err, localinstall.ErrClusterStart):
		reason, r.Detail = "start", "could not start"
		r.Fix = lastLines(out, 5) + "\nIf a port is in use, close that program and run again.\n" +
			"Otherwise delete the cluster: ~/.testkube/bin/kind delete cluster --name " + localinstall.ClusterName
	case errors.Is(err, localinstall.ErrClusterStorage):
		reason, r.Detail = "storage", "could not set up storage"
		r.Fix = lastLines(out, 5) + "\nRun again. If it keeps failing, delete the cluster:\n" +
			"  ~/.testkube/bin/kind delete cluster --name " + localinstall.ClusterName
	case errors.Is(err, localinstall.ErrClusterInspect):
		reason, r.Detail = "inspect", "could not read existing cluster"
		r.Fix = withWhy(out, "Check that Docker is running, then run again")
	case errors.Is(err, localinstall.ErrClusterKubeconfig):
		reason, r.Detail = "kubeconfig", "could not save its settings"
		r.Fix = withWhy(out, "Check that ~/.testkube is writable, then run again")
	case errors.As(err, &taken):
		reason, r.Detail = "port_taken", "port taken"
		r.Fix = fmt.Sprintf("Another program took port %d just now. Close it, then run again", taken.Port)
	default:
		why := lastLines(out, 5)
		if why == "" {
			why = err.Error()
		}
		r.Fix = why + "\nCheck that Docker has enough memory and disk, then run again"
	}
	printCheckResult(r)
	tracker.Send("install_local_failed", map[string]any{"stage": "cluster", "reason": reason})
	waitForEvents(tracker)
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

// Docker's own words first, when it gave any.
func withWhy(out, fix string) string {
	if why := lastLines(out, 3); why != "" {
		return why + "\n" + fix
	}
	return fix
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

// Two spaces minimum, so long values never touch the next column.
func column(s string) string {
	return s + strings.Repeat(" ", max(2, 16-utf8.RuneCountInString(s)))
}

func printCheckResult(r localinstall.Result) {
	icon := map[localinstall.Status]string{
		localinstall.StatusPass: ui.Green("✔"),
		localinstall.StatusWarn: ui.LightYellow("⚠"),
		localinstall.StatusFail: ui.LightRed("✖"),
	}[r.Status]
	// Padded before coloring: escape codes break widths.
	line := r.Detail
	switch {
	case r.Version != "":
		line = ui.LightGray(column(r.Version)) + r.Detail
		if r.Hint != "" {
			line += "  " + ui.LightGray(r.Hint)
		}
	case r.Hint != "":
		line = column(r.Detail) + ui.LightGray(r.Hint)
	}
	ui.Printf("  %s %-10s %s\n", icon, r.Name, line)
	if r.Fix == "" {
		return
	}
	for _, line := range strings.Split(r.Fix, "\n") {
		ui.Printf("      %s\n", line)
	}
}

// Sales sees trial installs through these; opt-outs send nothing.
func reportInstallLicenseEvent(tracker *telemetry.InstallTracker, license, event string) {
	if tracker.Enabled() {
		reportLicenseEvent(config.Data{TelemetryEnabled: true}, license, event)
	}
}

// os.Exit drops in-flight sends; give both a moment.
func waitForEvents(tracker *telemetry.InstallTracker) {
	tracker.Wait()
	waitLicenseEvents()
}

func printReady(tracker *telemetry.InstallTracker, ports localinstall.Ports) {
	url := fmt.Sprintf("http://localhost:%d", ports["dashboard"])
	label := func(s string) string { return ui.LightGray(fmt.Sprintf("%-12s", s)) }
	ui.NL()
	pterm.DefaultBox.WithBoxStyle(pterm.NewStyle(pterm.FgLightMagenta)).Println(
		ui.Green("✔") + " " + pterm.Bold.Sprint("Testkube is ready") + "\n\n" +
			"  " + label("Dashboard") + url + "\n" +
			"  " + label("Email") + localinstall.AdminEmail + "\n" +
			"  " + label("Password") + localinstall.AdminPassword)
	ui.Printf("\n  %s~/.testkube/bin/kind delete cluster --name %s\n", label("Remove it"), localinstall.ClusterName)
	opened := localinstall.OpenBrowser(url)
	if opened {
		ui.Printf("\n  Opening %s in your browser ...\n", url)
	} else {
		ui.Printf("\n  Open %s in your browser.\n", url)
	}
	tracker.Send("install_local_ready", map[string]any{"browser_opened": opened})
}
