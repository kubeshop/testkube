package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewPurgeCmd() *cobra.Command {
	var name, namespace string
	var yes bool

	cmd := &cobra.Command{
		Use:     "purge",
		Short:   "Uninstall Testkube from your current kubectl context",
		Long:    `Uninstall Testkube from your current kubectl context`,
		Aliases: []string{"uninstall"},
		Run: func(cmd *cobra.Command, args []string) {
			if !shouldPurge(name, namespace, yes, ui.Confirm) {
				ui.Info("Purge cancelled")
				return
			}

			originalVerbose := ui.Verbose
			ui.Verbose = true

			_, err := process.Execute("helm", "uninstall", "--namespace", namespace, name)
			ui.PrintOnError("uninstalling testkube", err)

			ui.Verbose = originalVerbose

		},
	}

	cmd.Flags().StringVar(&name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "namespace from where to uninstall")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")

	return cmd
}

// shouldPurge warns about the data loss and asks for confirmation, unless it was already given with --yes.
func shouldPurge(name, namespace string, yes bool, confirm func(string) bool) bool {
	if yes {
		return true
	}

	ui.Warn(fmt.Sprintf("This will uninstall the Testkube release %q from namespace %q.", name, namespace))
	ui.Warn("All Testkube data will be removed, including tests, test suites, triggers, webhooks and execution results.")
	ui.NL()

	return confirm("Do you want to continue?")
}
