package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/localinstall"
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
	// The install's Docker check, so the same fixes apply.
	if docker := localinstall.NewChecker().CheckDocker(cmd.Context()); docker.Status == localinstall.StatusFail {
		printCheckResult(docker)
		os.Exit(1)
	}
	cluster, err := localinstall.NewCluster()
	if err != nil {
		failUninstall(localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not start", Fix: err.Error()})
	}
	found, out, err := cluster.Remove(cmd.Context())
	switch {
	case errors.Is(err, localinstall.ErrClusterNotOurs):
		failUninstall(localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "not ours",
			Fix: fmt.Sprintf("%q wasn't created by this installer, so it's left alone.\n"+
				"Remove it yourself if you don't need it: ~/.testkube/bin/kind delete cluster --name %s", localinstall.ClusterName, localinstall.ClusterName)})
	case err != nil:
		failUninstall(localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not remove",
			Fix: withWhy(out, "Try: ~/.testkube/bin/kind delete cluster --name "+localinstall.ClusterName)})
	}
	if found {
		printCheckResult(localinstall.Result{Name: "cluster", Status: localinstall.StatusPass, Detail: "removed",
			Hint: fmt.Sprintf("%q", localinstall.ClusterName)})
	}
	removed, err := localinstall.RemoveFiles()
	if err != nil {
		failUninstall(localinstall.Result{Name: "files", Status: localinstall.StatusFail, Detail: "could not remove", Fix: err.Error()})
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
			failUninstall(localinstall.Result{Name: "data", Status: localinstall.StatusFail, Detail: "could not delete", Fix: fix})
		}
		if dataDeleted {
			printCheckResult(localinstall.Result{Name: "data", Status: localinstall.StatusPass, Detail: "deleted", Hint: "~/.testkube/data"})
		}
	}
	if !found && !removed && !dataDeleted {
		ui.Printf("  Nothing to uninstall: no local Testkube found.\n")
		return
	}
	ui.Printf("\n  Testkube is uninstalled.\n")
	if localinstall.HasData() {
		ui.Printf("  Your data is kept in ~/.testkube/data. `testkube install local` uses it again.\n" +
			"  To delete it too: testkube uninstall local --delete-data\n")
	}
}

func failUninstall(r localinstall.Result) {
	printCheckResult(r)
	os.Exit(1)
}
