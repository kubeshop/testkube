package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

// Under purge, so `uninstall local` never purges the real cluster.
func NewUninstallLocalCmd() *cobra.Command {
	var deleteData, yes bool
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Remove the local Testkube trial",
		Long: "Deletes the cluster and its settings. Your data in ~/.testkube/data is kept,\n" +
			"so installing again brings back your tests and results.",
		Example: "  " + uninstallCommand + "\n  " + uninstallCommand + " --delete-data",
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runUninstallLocal(cmd, deleteData, yes)
		},
	}
	cmd.Flags().BoolVar(&deleteData, "delete-data", false, "also delete your Testkube data in ~/.testkube/data")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	// The parent is purge, but users type uninstall.
	cmd.SetUsageTemplate(strings.Replace(cmd.UsageTemplate(), "{{.UseLine}}", uninstallCommand+" [flags]", 1))
	hideGlobalFlags(cmd)
	return cmd
}

const uninstallCommand = "testkube uninstall local"

var (
	errNeedsYes     = errors.New("--delete-data needs a yes, and there's no terminal to ask")
	errWindowsTrial = errors.New("the local trial lives inside WSL")
)

// Deleting data can't be undone, so no answer means no.
func shouldDeleteData(yes, interactive bool, confirm func() bool) (bool, error) {
	switch {
	case yes:
		return true, nil
	case !interactive:
		return false, errNeedsYes
	}
	return confirm(), nil
}

func confirmDataDeletion() bool {
	ui.Printf("  This also deletes your Testkube data in ~/.testkube/data\n" +
		"  (workflows, results, users). This can't be undone.\n")
	ok, _ := pterm.DefaultInteractiveConfirm.WithDefaultValue(false).Show("  Continue?")
	return ok
}

func runUninstallLocal(cmd *cobra.Command, deleteData, yes bool) {
	// Install only runs in WSL, so its files live there.
	if runtime.GOOS == "windows" {
		common.ExitOnLocalError(cmd, "install_local_uninstall_failed", "platform", common.NewCLIError(
			common.TKErrLocalUnsupportedPlatform, "Error removing the local trial on Windows",
			"In your WSL terminal, run:\n  "+uninstallCommand, errWindowsTrial))
	}
	if deleteData {
		ok, err := shouldDeleteData(yes, ui.StdinIsInteractive(), confirmDataDeletion)
		if err != nil {
			common.ExitOnLocalError(cmd, "install_local_uninstall_failed", "confirmation", common.NewCLIError(
				common.TKErrInvalidRuntimeParameter, "Error deleting data without confirmation",
				"Run again with --yes:\n  "+uninstallCommand+" --delete-data --yes", err))
		}
		if !ok {
			ui.Printf("  Nothing was removed.\n")
			return
		}
	}
	// Rerunning without it would silently keep the data.
	rerun := uninstallCommand
	if deleteData {
		rerun += " --delete-data"
	}
	tracker := newInstallTracker()
	tracker.OnError(segmentErrors(cmd, new(string)))
	start := time.Now()
	found, dockerHidden := false, false
	docker := localinstall.NewChecker().CheckDocker(cmd.Context())
	switch {
	case docker.Status != localinstall.StatusFail:
		found = removeCluster(cmd, tracker, rerun)
	// The cluster went with Docker; only our files are left.
	// Missing docker may only be hidden, e.g. WSL integration off.
	case docker.Detail == "not found":
		dockerHidden = true
		skipped := localinstall.Result{Name: "cluster", Status: localinstall.StatusWarn, Detail: "skipped",
			Hint: "Docker isn't reachable from here, so it was left alone"}
		if strings.Contains(docker.Fix, "WSL integration") {
			skipped.Fix = strings.Replace(docker.Fix, localinstall.InstallCommand, rerun, 1)
		}
		printCheckResult(skipped)
	// Removing the mark now would orphan a cluster still there.
	default:
		docker.Fix = strings.Replace(docker.Fix, localinstall.InstallCommand, rerun, 1)
		if docker.Detail != "permission denied" {
			docker.Fix = "The cluster may still be there. Start Docker, then run again:\n  " + rerun
		}
		failUninstall(cmd.Context(), tracker, rerun, strings.ReplaceAll(docker.Detail, " ", "_"), docker)
	}
	removed, err := localinstall.RemoveFiles(dockerHidden)
	if err != nil {
		failUninstall(cmd.Context(), tracker, rerun, "remove", localinstall.Result{Name: "files", Status: localinstall.StatusFail, Detail: "could not remove", Fix: err.Error()})
	}
	if removed {
		hint := "~/.testkube (cluster settings and caches)"
		if dockerHidden {
			hint = "~/.testkube (caches)"
		}
		printCheckResult(localinstall.Result{Name: "files", Status: localinstall.StatusPass, Detail: "removed", Hint: hint})
	}
	cliOut := !dockerHidden && common.DisconnectLocalCLI()
	if cliOut {
		printCheckResult(localinstall.Result{Name: "cli", Status: localinstall.StatusPass, Detail: "logged out", Hint: "of the local install"})
	}
	dataDeleted := false
	if deleteData && dockerHidden {
		printCheckResult(localinstall.Result{Name: "data", Status: localinstall.StatusWarn, Detail: "kept", Hint: "~/.testkube/data",
			Fix: "The cluster may still use it."})
	}
	if deleteData && !dockerHidden {
		var sudo string
		dataDeleted, sudo, err = localinstall.DeleteData(cmd.Context())
		if err != nil {
			fix := err.Error()
			if sudo != "" {
				fix = "Some files belong to the containers. Delete them with:\n  " + sudo
			}
			reason := "delete"
			if sudo != "" {
				reason = "needs_sudo"
			}
			failUninstall(cmd.Context(), tracker, rerun, reason, localinstall.Result{Name: "data", Status: localinstall.StatusFail, Detail: "could not delete", Fix: fix})
		}
		if dataDeleted {
			printCheckResult(localinstall.Result{Name: "data", Status: localinstall.StatusPass, Detail: "deleted", Hint: "~/.testkube/data"})
		}
	}
	tracker.Send("install_local_uninstall_done", map[string]any{"cluster_found": found, "files_removed": removed,
		"data_deleted": dataDeleted, "cli_logged_out": cliOut, "duration_s": int(time.Since(start).Seconds())})
	defer waitForEvents(tracker)
	if !found && !removed && !dataDeleted && !cliOut {
		if !localinstall.HasData() {
			ui.Printf("  Nothing to uninstall: no local Testkube found.\n")
			return
		}
		ui.Printf("  Testkube is already uninstalled.\n")
		printKeptData("To delete it")
		return
	}
	if dockerHidden {
		ui.Printf("\n  Not finished. Once Docker is reachable, run again:\n    %s\n", rerun)
		return
	}
	ui.Printf("\n  Testkube is uninstalled.\n")
	printKeptData("To delete it too")
}

// A second run must still point at kept data.
func printKeptData(lead string) {
	if localinstall.HasData() {
		ui.Printf("  Your data is kept in ~/.testkube/data. Installing again reuses it.\n"+
			"  %s:\n    %s --delete-data\n", lead, uninstallCommand)
	}
}

func removeCluster(cmd *cobra.Command, tracker *telemetry.InstallTracker, rerun string) bool {
	cluster, err := localinstall.NewCluster()
	if err != nil {
		failUninstall(cmd.Context(), tracker, rerun, "setup", localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not start", Fix: err.Error()})
	}
	found, out, err := cluster.Remove(cmd.Context())
	var elsewhere localinstall.ClusterElsewhereError
	switch {
	case errors.As(err, &elsewhere):
		failUninstall(cmd.Context(), tracker, rerun, "elsewhere", localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "elsewhere",
			Fix: fmt.Sprintf("It's in Docker %q, but %q is active now.\n"+
				"Switch to it, then run again:\n  docker context use %s\n  %s", elsewhere.In, elsewhere.Active, elsewhere.In, rerun)})
	case errors.Is(err, localinstall.ErrClusterNotOurs):
		failUninstall(cmd.Context(), tracker, rerun, "not_ours", localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "not ours",
			Fix: fmt.Sprintf("%q wasn't created by this installer, so it's left alone.\n"+
				"Remove it yourself if you don't need it:\n  ~/.testkube/bin/kind delete cluster --name %s", localinstall.ClusterName, localinstall.ClusterName)})
	case err != nil:
		failUninstall(cmd.Context(), tracker, rerun, "remove", localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not remove",
			Fix: withWhy(out, "Remove it yourself:\n  ~/.testkube/bin/kind delete cluster --name "+localinstall.ClusterName)})
	}
	if found {
		printCheckResult(localinstall.Result{Name: "cluster", Status: localinstall.StatusPass, Detail: "removed",
			Hint: fmt.Sprintf("%q", localinstall.ClusterName)})
	}
	return found
}

func failUninstall(ctx context.Context, tracker *telemetry.InstallTracker, rerun, reason string, r localinstall.Result) {
	// Ctrl+C cancels the Docker call: a choice, not a failure.
	if ctx.Err() != nil {
		ui.Printf("  Uninstall cancelled. To finish, run again:\n    %s\n", rerun)
		tracker.Send("install_local_uninstall_aborted", map[string]any{"stage": r.Name})
		waitForEvents(tracker)
		os.Exit(130)
	}
	printCheckResult(r)
	tracker.Send("install_local_uninstall_failed", map[string]any{"stage": r.Name, "reason": reason})
	tracker.Error(string(common.TKErrLocalUninstallFailed), "uninstall_"+r.Name, reason)
	waitForEvents(tracker)
	os.Exit(1)
}
