package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

// Under purge, so `uninstall local` never purges the real cluster.
func NewUninstallLocalCmd() *cobra.Command {
	var deleteData, yes bool
	cmd := &cobra.Command{
		Use:    "local",
		Short:  "Remove the local Testkube trial made by install local",
		Hidden: true,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runUninstallLocal(cmd, deleteData, yes)
		},
	}
	cmd.Flags().BoolVar(&deleteData, "delete-data", false, "also delete your Testkube data in ~/.testkube/data")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	return cmd
}

var errNeedsYes = errors.New("--delete-data needs confirmation. Run again with --yes")

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
	if deleteData {
		ok, err := shouldDeleteData(yes, ui.StdinIsInteractive(), confirmDataDeletion)
		if err != nil {
			ui.Printf("  %s\n", err)
			os.Exit(1)
		}
		if !ok {
			ui.Printf("  Nothing was removed.\n")
			return
		}
	}
	tracker := newInstallTracker()
	start := time.Now()
	found := false
	docker := localinstall.NewChecker().CheckDocker(cmd.Context())
	switch {
	case docker.Status != localinstall.StatusFail:
		found = removeCluster(cmd, tracker)
	// The cluster went with Docker; only our files are left.
	case docker.Detail == "not found":
		printCheckResult(localinstall.Result{Name: "cluster", Status: localinstall.StatusWarn, Detail: "skipped",
			Hint: "Docker isn't installed, so there's no cluster left"})
	// Removing the mark now would orphan a cluster still there.
	default:
		docker.Fix = strings.Replace(docker.Fix, "Then run the installer again", "Then run `testkube uninstall local` again", 1)
		if docker.Detail != "permission denied" {
			docker.Fix = "The cluster may still be there. Start Docker, then run `testkube uninstall local` again."
		}
		failUninstall(tracker, strings.ReplaceAll(docker.Detail, " ", "_"), docker)
	}
	removed, err := localinstall.RemoveFiles()
	if err != nil {
		failUninstall(tracker, "remove", localinstall.Result{Name: "files", Status: localinstall.StatusFail, Detail: "could not remove", Fix: err.Error()})
	}
	if removed {
		printCheckResult(localinstall.Result{Name: "files", Status: localinstall.StatusPass, Detail: "removed",
			Hint: "~/.testkube (cluster settings and caches)"})
	}
	dataDeleted := false
	if deleteData {
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
			failUninstall(tracker, reason, localinstall.Result{Name: "data", Status: localinstall.StatusFail, Detail: "could not delete", Fix: fix})
		}
		if dataDeleted {
			printCheckResult(localinstall.Result{Name: "data", Status: localinstall.StatusPass, Detail: "deleted", Hint: "~/.testkube/data"})
		}
	}
	tracker.Send("install_local_uninstall_done", map[string]any{"cluster_found": found, "files_removed": removed,
		"data_deleted": dataDeleted, "duration_s": int(time.Since(start).Seconds())})
	defer waitForEvents(tracker)
	if !found && !removed && !dataDeleted {
		if !localinstall.HasData() {
			ui.Printf("  Nothing to uninstall: no local Testkube found.\n")
			return
		}
		ui.Printf("  Testkube is already uninstalled.\n")
		printKeptData("To delete it")
		return
	}
	ui.Printf("\n  Testkube is uninstalled.\n")
	printKeptData("To delete it too")
}

// A second run must still point at kept data.
func printKeptData(lead string) {
	if localinstall.HasData() {
		ui.Printf("  Your data is kept in ~/.testkube/data. `testkube install local` uses it again.\n"+
			"  %s: testkube uninstall local --delete-data\n", lead)
	}
}

func removeCluster(cmd *cobra.Command, tracker *telemetry.InstallTracker) bool {
	cluster, err := localinstall.NewCluster()
	if err != nil {
		failUninstall(tracker, "setup", localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not start", Fix: err.Error()})
	}
	found, out, err := cluster.Remove(cmd.Context())
	switch {
	case errors.Is(err, localinstall.ErrClusterNotOurs):
		failUninstall(tracker, "not_ours", localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "not ours",
			Fix: fmt.Sprintf("%q wasn't created by this installer, so it's left alone.\n"+
				"Remove it yourself if you don't need it: ~/.testkube/bin/kind delete cluster --name %s", localinstall.ClusterName, localinstall.ClusterName)})
	case err != nil:
		failUninstall(tracker, "remove", localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not remove",
			Fix: withWhy(out, "Try: ~/.testkube/bin/kind delete cluster --name "+localinstall.ClusterName)})
	}
	if found {
		printCheckResult(localinstall.Result{Name: "cluster", Status: localinstall.StatusPass, Detail: "removed",
			Hint: fmt.Sprintf("%q", localinstall.ClusterName)})
	}
	return found
}

func failUninstall(tracker *telemetry.InstallTracker, reason string, r localinstall.Result) {
	printCheckResult(r)
	tracker.Send("install_local_uninstall_failed", map[string]any{"stage": r.Name, "reason": reason})
	waitForEvents(tracker)
	os.Exit(1)
}
