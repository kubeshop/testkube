package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/ui"
)

// Under purge, so `uninstall local` never purges the real cluster.
func NewUninstallLocalCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "local",
		Short:  "Remove the local Testkube trial made by install local",
		Hidden: true,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runUninstallLocal(cmd)
		},
	}
}

func runUninstallLocal(cmd *cobra.Command) {
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
				"Remove it yourself if you don't need it: kind delete cluster --name %s", localinstall.ClusterName, localinstall.ClusterName)})
	case err != nil:
		failUninstall(localinstall.Result{Name: "cluster", Status: localinstall.StatusFail, Detail: "could not remove",
			Fix: withWhy(out, "Try: ~/.testkube/bin/kind delete cluster --name "+localinstall.ClusterName)})
	case !found:
		ui.Printf("  Nothing to uninstall: no local Testkube found.\n")
		return
	}
	printCheckResult(localinstall.Result{Name: "cluster", Status: localinstall.StatusPass, Detail: "removed",
		Hint: fmt.Sprintf("%q", localinstall.ClusterName)})
}

func failUninstall(r localinstall.Result) {
	printCheckResult(r)
	os.Exit(1)
}
