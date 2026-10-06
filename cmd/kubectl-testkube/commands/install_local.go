package commands

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewInstallLocalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "local",
		Short: "Install Testkube on this machine",
		// Hidden until the guided installer is complete.
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			checker := localinstall.NewChecker()
			results := checker.CheckTools(cmd.Context())
			for _, r := range results {
				printCheckResult(r)
			}
			if localinstall.HasFailure(results) {
				os.Exit(1)
			}
			for _, r := range checker.CheckMachine(cmd.Context()) {
				printCheckResult(r)
			}
		},
	}
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
