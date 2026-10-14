package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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
		Long: "Sets up a Testkube trial in its own cluster inside Docker.\n" +
			"Needs Docker and the license key from your trial email.\n" +
			"The first run takes 5 to 10 minutes. Your own tools and clusters are not changed.\n\n" +
			ownClusterHint + "\n\n" +
			"To remove it later:\n  " + uninstallCommand,
		Example: "  " + localinstall.InstallCommand + "\n  " + localinstall.InstallCommand + " --license <key>",
		// Hidden until the guided installer is complete.
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			printBanner()
			existing, running := localinstall.ExistingTrial(cmd.Context())
			kubeContext, kubeContextLocal := localinstall.CurrentKubeContext()
			// Reruns already chose; scripts and kubectl-less machines can't.
			askPath := existing == localinstall.TrialAbsent && kubeContext != "" && ui.StdinIsInteractive()
			if !askPath {
				ui.Printf("%s\n", ui.LightGray(localInstallLead+"\n"+ownClusterHint))
			}
			tracker := newInstallTracker()
			var key string
			tracker.OnError(segmentErrors(cmd, &key))
			if tracker.Enabled() {
				ui.Printf("\n%s\n", ui.LightGray(telemetry.InstallNotice))
			}
			tracker.Send("install_local_started", nil)
			exitIfCancelled(cmd, tracker, "path")
			if askPath && askOwnCluster(tracker, kubeContext, kubeContextLocal) {
				runInitDemo(tracker)
				return
			}
			printPlan(existing, running)
			exitIfCancelled(cmd, tracker, "plan")

			printStep(1, "Checking your license")
			key = runLicenseStep(tracker, licenseKey)
			// The login needs it; validate already sent the key there.
			owner := telemetry.GetEmail(key)
			if tracker.Enabled() {
				tracker.Identify(owner)
			}
			email := localinstall.RememberLoginEmail(owner)
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
				if r.Alternative != "" {
					r.Fix += "\n" + r.Alternative
				}
				printCheckResult(r)
			}
			if localinstall.HasFailure(results) {
				trackChecks(tracker, results, checker.Facts())
				tracker.Send("install_local_failed", map[string]any{"stage": "tools"})
				tracker.Error(string(common.TKErrLocalToolsFailed), "tools", failedChecks(results))
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
			runTestkubeStep(cmd, tracker, key, email, ports)
			cli, err := common.ConnectLocalCLI(cmd.Context(), ports, email, false)
			if err != nil {
				ui.Debug("testkube CLI login failed", err.Error())
			}
			printReady(tracker, ports, email, cli, common.LocalCLIFailure(err))
			reportInstallLicenseEvent(tracker, key, licensevalidator.EventCLIInstallFinished)
			waitForEvents(tracker)
		},
	}
	cmd.Flags().StringVarP(&licenseKey, "license", "l", "", "Testkube license key from your trial email (or set TESTKUBE_LICENSE)")
	hideGlobalFlags(cmd)
	return cmd
}

// The API flags do nothing here; --namespace would mislead.
func hideGlobalFlags(cmd *cobra.Command) {
	// Help and flag errors both render through usage.
	usage := cmd.UsageFunc()
	cmd.SetUsageFunc(func(c *cobra.Command) error {
		c.InheritedFlags().VisitAll(func(f *pflag.Flag) { f.Hidden = true })
		return usage(c)
	})
}

func runLicenseStep(tracker *telemetry.InstallTracker, flagKey string) string {
	// curl | bash can't pass flags easily; a variable can.
	if flagKey == "" {
		flagKey = os.Getenv("TESTKUBE_LICENSE")
	}
	if flagKey == "" && !ui.StdinIsInteractive() {
		failLicense(tracker, localinstall.Result{Name: "license", Status: localinstall.StatusFail, Detail: "no key given",
			Fix: localinstall.LicenseHelp + "\nThen run again:\n  " + localinstall.InstallCommand + " --license <key>"})
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
	then := "\nThen run again:\n  " + localinstall.InstallCommand
	r := localinstall.Result{Name: "license", Status: localinstall.StatusFail, Fix: localinstall.LicenseHelp + then}
	switch {
	case errors.Is(err, localinstall.ErrLicenseUnreachable):
		r.Detail, r.Fix = "cannot reach license.testkube.io", localinstall.LicenseUnreachableHelp
	case errors.Is(err, localinstall.ErrLicenseMissing):
		r.Detail = "no key entered"
	case errors.Is(err, localinstall.ErrLicenseInvalid):
		r.Detail = "not valid"
	case errors.Is(err, localinstall.ErrLicenseExpired):
		r.Detail, r.Fix = "expired", "This key has expired. Get a new one at https://testkube.io/get-started/on-prem"+then
		if !expiry.IsZero() {
			r.Detail = "expired on " + expiry.Local().Format("2 Jan 2006")
		}
	default:
		r.Detail, r.Fix = "could not read the key", "Pass it as a flag instead:\n  "+localinstall.InstallCommand+" --license <key>"
	}
	return r
}

// Segment holds every CLI error; PostHog keeps the install journey.
func segmentErrors(cmd *cobra.Command, license *string) func(code, stage, reason string) {
	return func(code, stage, reason string) {
		// Fixed words only: raw errors can carry paths or keys.
		_, _ = telemetry.SendCmdErrorEventWithLicense(cmd, common.Version, stage,
			stage+": "+reason, *license, stage, code)
	}
}

func failedChecks(results []localinstall.Result) string {
	var failed []string
	for _, r := range results {
		if r.Status == localinstall.StatusFail {
			failed = append(failed, r.Name+" "+r.Detail)
		}
	}
	return strings.Join(failed, ", ")
}

func failLicense(tracker *telemetry.InstallTracker, r localinstall.Result) {
	printCheckResult(r)
	tracker.Send("install_local_failed", map[string]any{"stage": "license"})
	// "expired on <date>" would split grouping by date.
	reason, _, _ := strings.Cut(r.Detail, " on ")
	tracker.Error(string(common.TKErrLocalLicenseFailed), "license", reason)
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
	return common.NewInstallTracker()
}

// Ctrl+C is the user quitting, not a step failing.
func exitIfCancelled(cmd *cobra.Command, tracker *telemetry.InstallTracker, stage string) {
	if cmd.Context().Err() == nil {
		return
	}
	abortInstall(tracker, stage)
}

var cancelNotice sync.Once

// Before any wait, or Ctrl+C looks ignored; once, from either path.
func printCancelled(stage string) {
	kept := ""
	// Tools, node image and Testkube images; reruns skip finished ones.
	if stage == "tool_install" || stage == "cluster" || stage == "testkube" {
		kept = " Finished downloads are kept."
	}
	cancelNotice.Do(func() {
		ui.Printf("\n  Install cancelled.%s\n  To finish, run again:\n    %s\n", kept, localinstall.InstallCommand)
	})
}

func abortInstall(tracker *telemetry.InstallTracker, stage string) {
	printCancelled(stage)
	tracker.Send("install_local_aborted", map[string]any{"stage": stage})
	waitForEvents(tracker)
	os.Exit(130)
}

// The name guards against installing into a work cluster.
func askOwnCluster(tracker *telemetry.InstallTracker, kubeContext string, local bool) bool {
	if !local {
		kubeContext += " (remote)"
	}
	thisMachine := "On this machine           Its own cluster in Docker (recommended)"
	ownCluster := "In my Kubernetes cluster  Current kubectl context: " + kubeContext
	ui.NL()
	choice, err := pterm.DefaultInteractiveSelect.
		WithOptions([]string{thisMachine, ownCluster}).
		WithDefaultOption(thisMachine).
		WithFilter(false).
		WithOnInterruptFunc(func() { abortInstall(tracker, "path") }).
		Show("Choose where Testkube runs")
	own := err == nil && choice == ownCluster
	path := "docker"
	if own {
		path = "cluster"
	}
	tracker.Send("install_local_path", map[string]any{"path": path, "context_local": local})
	return own
}

// Never -y: its confirm names the target cluster.
func runInitDemo(tracker *telemetry.InstallTracker) {
	waitForEvents(tracker)
	self, err := os.Executable()
	if err == nil {
		// Own process: init demo needs the root command's flags.
		// No context: Ctrl+C reaches it directly; a kill skips cleanup.
		demo := exec.Command(self, "init", "demo")
		demo.Stdin, demo.Stdout, demo.Stderr = os.Stdin, os.Stdout, os.Stderr
		err = demo.Run()
	}
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr) && exitErr.ExitCode() < 0:
		os.Exit(130)
	case errors.As(err, &exitErr):
		os.Exit(exitErr.ExitCode())
	case err != nil:
		ui.Printf("\nTo install Testkube in your cluster, run:\n  %s\n", localinstall.InitDemoCommand)
		os.Exit(1)
	}
}

func printBanner() {
	pterm.DefaultBox.WithBoxStyle(pterm.NewStyle(pterm.FgLightMagenta)).Println(
		pterm.Bold.Sprint("Testkube On-Prem Installer") + "  " + ui.LightGray(common.Version) +
			"\nTry Testkube on your own machine.")
}

const (
	localInstallLead = "This sets up Testkube locally, in its own cluster inside Docker."
	ownClusterHint   = "Already have a Kubernetes cluster? To install Testkube there, run:\n  " + localinstall.InitDemoCommand
)

// Numbered like the step headers, so [3/6] means line 3.
func printPlan(existing localinstall.ExistingState, running bool) {
	plan := []string{"check your license key", "check the tools it needs", "check this machine",
		"download helm and kind into ~/.testkube/bin (kubectl too, if missing)",
		"create a local cluster called \"testkube\" in Docker", "install Testkube and open it in your browser"}
	note := "First run takes 5 to 10 minutes, mostly downloads. Your own tools and clusters are not changed."
	// Reruns download nothing; promising minutes would mislead.
	if existing != localinstall.TrialAbsent {
		plan[3], plan[4] = "check helm and kind in ~/.testkube/bin", "start the local cluster \"testkube\" in Docker"
	}
	switch existing {
	case localinstall.TrialSetUp:
		plan[5] = "update Testkube and open it in your browser"
		note = "Testkube is already set up here. This run updates it, usually in under a minute.\nYour data is kept."
		if running {
			note += " To only check it, run:\n  " + statusLocalCommand
		}
	case localinstall.TrialUnfinished:
		note = "A previous install didn't finish. This run continues it.\nFinished downloads are kept."
	case localinstall.TrialNodeStopped:
		plan[5] = "install or update Testkube and open it in your browser"
		note = "Your local cluster is stopped. This run starts it and checks Testkube.\nFinished downloads are kept."
	}
	ui.Printf("\nThis installer will\n")
	for i, step := range plan {
		ui.Printf("  %d. %s\n", i+1, step)
	}
	ui.Printf("\n%s\n", ui.LightGray(note+"\nPress Ctrl+C to exit at any time."))
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
		r.Fix = "Install " + name + " yourself:\n  " + localinstall.ToolManualURL(name) + "\nThen run again:\n  " + localinstall.InstallCommand
	case errors.Is(err, localinstall.ErrChecksumMismatch):
		reason, r.Detail = "checksum", "download was damaged"
		r.Fix = "If it repeats, a proxy may be changing downloads.\n" + runAgain
	case errors.Is(err, localinstall.ErrSaveFailed):
		reason, r.Detail = "save", "could not save"
		r.Fix = err.Error() + "\nCheck that you can write to ~/.testkube/bin, " + thenRunAgain
	default:
		reason, r.Detail = "download", "could not download"
		r.Fix = err.Error() + "\nCheck your network, proxy or firewall, " + thenRunAgain
	}
	printCheckResult(r)
	tracker.Send("install_local_failed", map[string]any{"stage": "tool_install", "tool": name, "reason": reason})
	tracker.Error(string(common.TKErrLocalToolInstallFailed), "tool_install", name+" "+reason)
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

func runTestkubeStep(cmd *cobra.Command, tracker *telemetry.InstallTracker, license, email string, ports localinstall.Ports) {
	printStep(6, "Installing Testkube")
	secrets, err := localinstall.LoadOrCreateSecrets()
	if err != nil {
		failTestkube(tracker, "", err, "")
	}
	installer, err := localinstall.NewInstaller(common.Version)
	if err != nil {
		failTestkube(tracker, "", err, "")
	}
	var progress atomic.Pointer[localinstall.Progress]
	installer.OnProgress = func(p localinstall.Progress) { progress.Store(&p) }
	installer.LoginEmail = email
	start := time.Now()
	show := showPlain
	if canShowLive() {
		show = showLive
	}
	var stopped sync.Once
	stop := show(os.Stdout, start, &progress)
	finished := make(chan struct{})
	// Helm takes seconds to clean up; say so before it does.
	go func() {
		select {
		case <-cmd.Context().Done():
			stopped.Do(stop)
			printCancelled("testkube")
		case <-finished:
		}
	}()
	state, out, err := installer.Install(cmd.Context(), ports, secrets, license)
	close(finished)
	stopped.Do(stop)
	exitIfCancelled(cmd, tracker, "testkube")
	if err != nil {
		failTestkube(tracker, out, err, state.ReportPath)
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

// Hidden on a rerun: cached images download nothing.
func downloadedText(n int64) string {
	switch {
	case n < 1_000_000:
		return ""
	case n < localinstall.ImagesSize:
		return byteSize(n) + " of ~" + byteSize(localinstall.ImagesSize) + " · "
	}
	return byteSize(n) + " downloaded · "
}

func byteSize(n int64) string {
	if n < 1_000_000_000 {
		return fmt.Sprintf("%d MB", n/1_000_000)
	}
	return fmt.Sprintf("%.1f GB", float64(n)/1e9)
}

func took(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
}

func failTestkube(tracker *telemetry.InstallTracker, out string, err error, report string) {
	r := localinstall.Result{Name: "testkube", Status: localinstall.StatusFail, Version: localinstall.AppVersion, Detail: "could not install"}
	reason := "install"
	var stuck localinstall.StuckError
	isStuck := errors.As(err, &stuck)
	switch {
	case errors.Is(err, localinstall.ErrSecretsLost):
		reason, r.Detail = "secrets_lost", "old data can't be opened"
		r.Fix = "~/.testkube/data holds data from an earlier install whose passwords are gone.\n" +
			"Delete the cluster and that data to start fresh:\n" +
			"  testkube uninstall local --delete-data\n" +
			"Then run again:\n  " + localinstall.InstallCommand
	case errors.Is(err, localinstall.ErrSecretsDamaged):
		reason, r.Detail = "secrets_damaged", "saved passwords can't be read"
		r.Fix = "Restore ~/.testkube/secrets.json, or start fresh:\n  testkube uninstall local --delete-data"
	case errors.Is(err, localinstall.ErrStuck):
		reason, r.Detail = "stuck", "gave up after "+took(stuck.After)
		if errors.Is(err, localinstall.ErrRunnerInstall) {
			r.Name = "runner"
		}
	case errors.Is(err, localinstall.ErrRunnerInstall):
		reason, r.Name = "runner", "runner"
		r.Fix = withWhy(out, "Testkube is installed; only the runner failed.\n"+runAgain)
	case errors.Is(err, localinstall.ErrInstallTimeout):
		reason, r.Detail = "timeout", "timed out after 15 minutes"
		r.Fix = withWhy(out, "Finished downloads are kept.\n"+runAgain)
	case errors.Is(err, localinstall.ErrChartDownload):
		reason, r.Detail = "chart_download", "could not download"
		r.Fix = withWhy(out, "Check your network, proxy or firewall, "+thenRunAgain)
	case errors.Is(err, localinstall.ErrTestkubePrepare):
		reason, r.Detail = "prepare", "could not prepare the cluster"
		r.Fix = withWhy(out, runAgain)
	default:
		if out == "" {
			out = err.Error()
		}
		r.Fix = withWhy(out, runAgain)
	}
	if isStuck {
		var docker localinstall.DockerResources
		if stuck.Reason == "no_cpu" || stuck.Reason == "no_memory" {
			docker = localinstall.ReadDockerResources(context.Background())
		}
		r.Fix = stuckFix(stuck.Stuck, docker)
	}
	// Support asks for it later; no nudge now.
	if report != "" {
		r.Fix += "\n" + ui.LightGray("Details saved to "+homeRelative(report))
	}
	printCheckResult(r)
	props := map[string]any{"stage": "testkube", "reason": reason}
	if isStuck {
		props["stuck_service"], props["stuck_reason"] = stuck.Service, stuck.Reason
	}
	tracker.Send("install_local_failed", props)
	tracker.Error(string(common.TKErrLocalTestkubeInstallFailed), "testkube", reason)
	waitForEvents(tracker)
	os.Exit(1)
}

const (
	runAgain               = localinstall.RunAgain
	thenRunAgain           = localinstall.ThenRunAgain
	setContextLocalCommand = "testkube set context local"
	contactFix             = "If it fails again, contact us: https://testkube.io/contact"
	retryFix               = runAgain + "\n" + contactFix
	// A rerun rarely fixes these, so don't pretend it will.
	bugFix = "This looks like a Testkube problem, not your setup.\n" + retryFix
)

// Plain words for what's stuck; helm's own lines never say why.
func stuckFix(st localinstall.Stuck, docker localinstall.DockerResources) string {
	name := st.Title()
	switch st.Reason {
	case "downloading":
		return fmt.Sprintf("%s is still downloading its image %s\nYour connection may be slow. Finished downloads are kept.\n%s",
			name, st.Detail, runAgain)
	case "image_pull":
		return fmt.Sprintf("%s can't download its image %s\nFinished downloads are kept. Check your network, proxy or firewall, %s",
			name, st.Detail, thenRunAgain)
	// No numbers: they change. No docker login: cluster ignores it.
	case "rate_limit":
		return fmt.Sprintf("%s can't download its image %s\nDocker Hub's download limit was reached for this network.\n"+
			"Details: https://docs.docker.com/docker-hub/usage/\n"+
			"Finished downloads are kept. Wait a few hours or switch networks, %s", name, st.Detail, thenRunAgain)
	case "image_missing":
		return fmt.Sprintf("%s can't download its image %s\nIt doesn't exist or needs a login. %s", name, st.Detail, bugFix)
	case "no_cpu", "no_memory":
		return name + " can't start: " + resourceFix(st.Reason == "no_cpu", docker)
	case "oom":
		return fmt.Sprintf("%s ran out of memory (limit %s).\n%s", name, st.Detail, bugFix)
	case "storage":
		return fmt.Sprintf("%s can't get its storage:\n  %s\n%s", name, st.Detail, retryFix)
	case "config":
		return fmt.Sprintf("%s can't start:\n  %s\n%s", name, st.Detail, bugFix)
	case "crashloop", "job_failed":
		head := fmt.Sprintf("%s keeps crashing (restarted %d times).", name, st.Restarts)
		if st.Reason == "job_failed" {
			head = name + " failed."
		}
		if len(st.Logs) > 0 {
			head += " Last log lines:\n  " + strings.Join(st.Logs, "\n  ")
		}
		return head + "\n" + bugFix
	case "not_ready":
		return fmt.Sprintf("%s is %s.\n%s", name, st.Detail, retryFix)
	}
	return fmt.Sprintf("%s can't start:\n  %s\n%s", name, st.Detail, retryFix)
}

// Each Docker app sets its size in a different place.
func resourceFix(cpu bool, docker localinstall.DockerResources) string {
	have := fmt.Sprintf("Docker has %s of memory, Testkube needs %d GB.", docker.MemoryText(), localinstall.NeededMemoryGB)
	desktop := fmt.Sprintf("set Memory to %d GB or more", localinstall.NeededMemoryGB)
	colima := fmt.Sprintf("--memory %d", localinstall.NeededMemoryGB)
	other := "Free up memory or use a machine with more"
	if cpu {
		have = fmt.Sprintf("Docker has %d CPUs, Testkube needs %d.", docker.CPUs, localinstall.NeededCPUs)
		desktop = fmt.Sprintf("set CPUs to %d or more", localinstall.NeededCPUs)
		colima = fmt.Sprintf("--cpu %d", localinstall.NeededCPUs)
		other = "Use a machine with more CPUs"
	}
	then := ", " + thenRunAgain
	switch {
	// Docker is big enough; something else in the cluster took it.
	case docker.Enough(cpu):
		if cpu {
			return "there isn't enough free CPU in the cluster for it.\n" + retryFix
		}
		return "there isn't enough free memory in the cluster for it.\n" + retryFix
	case docker.Engine == "":
		if cpu {
			return fmt.Sprintf("Docker doesn't have enough CPU.\nGive Docker %d CPUs or more%s", localinstall.NeededCPUs, then)
		}
		return fmt.Sprintf("Docker doesn't have enough memory.\nGive Docker %d GB or more%s", localinstall.NeededMemoryGB, then)
	case docker.DockerDesktop():
		return have + "\nOpen Docker Desktop > Settings > Resources, " + desktop + then
	case docker.Colima():
		return have + "\nRestart Colima with more:\n  colima stop && colima start " + colima + "\nThen run again:\n  " + localinstall.InstallCommand
	}
	return have + "\n" + other + then
}

// Short, and the same on every machine support talks to.
func homeRelative(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || !strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return path
	}
	return "~" + strings.TrimPrefix(path, home)
}

// pterm's light white text vanishes on light terminals.
// Removed when done: its success line clashes with our rows.
func startSpinner(text string) *pterm.SpinnerPrinter {
	// pterm can't redraw a wrapped line; room for icon and timer.
	text = fit(text, pterm.GetTerminalWidth()-15)
	spinner, _ := pterm.DefaultSpinner.
		WithSequence(` ⠋ `, ` ⠹ `, ` ⠼ `, ` ⠦ `, ` ⠇ `).
		WithMessageStyle(pterm.NewStyle(pterm.FgDefault)).
		WithRemoveWhenDone(true).
		Start(text)
	return spinner
}

// Status shows it too, so the wording lives once.
func nameTakenFix() string {
	return fmt.Sprintf("A kind cluster named %q exists that this installer didn't create.\n"+
		"Delete it if you don't need it:\n  ~/.testkube/bin/kind delete cluster --name %s\nThen run again:\n  %s",
		localinstall.ClusterName, localinstall.ClusterName, localinstall.InstallCommand)
}

func failCluster(tracker *telemetry.InstallTracker, out string, err error) {
	r := localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not create"}
	reason := "create"
	var taken localinstall.PortTakenError
	switch {
	case errors.Is(err, localinstall.ErrClusterNotOurs):
		reason, r.Detail = "not_ours", "name already taken"
		r.Fix = nameTakenFix()
	case errors.Is(err, localinstall.ErrClusterStart):
		reason, r.Detail = "start", "could not start"
		r.Fix = lastLines(out, 5) + "\nIf a port is in use, close that program, " + thenRunAgain + "\n" +
			"Otherwise remove it:\n  testkube uninstall local"
	case errors.Is(err, localinstall.ErrClusterStorage):
		reason, r.Detail = "storage", "could not set up storage"
		r.Fix = lastLines(out, 5) + "\n" + runAgain + "\nIf it keeps failing, remove it:\n" +
			"  testkube uninstall local"
	case errors.Is(err, localinstall.ErrClusterInspect):
		reason, r.Detail = "inspect", "could not read existing cluster"
		r.Fix = withWhy(out, "Check that Docker is running, "+thenRunAgain)
	case errors.Is(err, localinstall.ErrClusterKubeconfig):
		reason, r.Detail = "kubeconfig", "could not save its settings"
		r.Fix = withWhy(out, "Check that ~/.testkube is writable, "+thenRunAgain)
	case errors.As(err, &taken):
		reason, r.Detail = "port_taken", "port taken"
		r.Fix = fmt.Sprintf("Another program took port %d just now. Close it, %s", taken.Port, thenRunAgain)
	default:
		why := lastLines(out, 5)
		if why == "" {
			why = err.Error()
		}
		r.Fix = why + "\nCheck that Docker has enough memory and disk, " + thenRunAgain
	}
	printCheckResult(r)
	tracker.Send("install_local_failed", map[string]any{"stage": "cluster", "reason": reason})
	tracker.Error(string(common.TKErrLocalClusterFailed), "cluster", reason)
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
		localinstall.StatusWarn: ui.Yellow("⚠"),
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

func printReady(tracker *telemetry.InstallTracker, ports localinstall.Ports, email string, cli common.LocalCLIState, cliFailure string) {
	url := fmt.Sprintf("http://localhost:%d", ports["dashboard"])
	label := func(s string) string { return ui.LightGray(fmt.Sprintf("%-12s", s)) }
	ui.NL()
	pterm.DefaultBox.WithBoxStyle(pterm.NewStyle(pterm.FgLightMagenta)).Println(
		ui.Green("✔") + " " + pterm.Bold.Sprint("Testkube is ready") + "\n\n" +
			"  " + label("Dashboard") + url + "\n" +
			"  " + label("Email") + email + "\n" +
			"  " + label("Password") + localinstall.AdminPassword)
	switch cli {
	case common.LocalCLIConnected:
		ui.Printf("\n  %s testkube CLI now uses this local install\n", ui.Green("✔"))
	case common.LocalCLIOtherLogin:
		ui.Printf("\n  %s testkube CLI still uses your other Testkube login\n"+
			"      To use this local install instead, run:\n        %s\n", ui.LightGray("○"), setContextLocalCommand)
	default:
		ui.Printf("\n  %s testkube CLI couldn't log in to this install\n"+
			"      To try again, run:\n        %s\n", ui.LightGray("○"), setContextLocalCommand)
	}
	ui.Printf("\n  To see this again:\n    %s\n", statusLocalCommand)
	ui.Printf("\n  To remove it later:\n    %s\n", uninstallCommand)
	opened := localinstall.OpenBrowser(url)
	if opened {
		ui.Printf("\n  Opening %s in your browser...\n", url)
	} else {
		ui.Printf("\n  Open %s in your browser.\n", url)
	}
	tracker.Send("install_local_ready", map[string]any{"browser_opened": opened, "cli_connected": cli == common.LocalCLIConnected, "cli_failure": cliFailure})
}
