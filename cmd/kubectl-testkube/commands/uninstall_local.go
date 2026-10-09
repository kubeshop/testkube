package commands

import (
	"github.com/spf13/cobra"

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
			ui.Printf("Uninstalling the local trial is not ready yet.\n")
		},
	}
}
