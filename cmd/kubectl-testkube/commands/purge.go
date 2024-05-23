package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewPurgeCmd() *cobra.Command {
	var name, namespace string

	cmd := &cobra.Command{
		Use:     "purge",
		Short:   "Uninstall Testkube from your current kubectl context",
		Long:    `Uninstall Testkube from your current kubectl context`,
		Aliases: []string{"uninstall"},
		Run: func(cmd *cobra.Command, args []string) {
			originalVerbose := ui.Verbose
			ui.Verbose = true

			_, err := process.Execute("helm", "uninstall", "--namespace", namespace, name)
			ui.PrintOnError("uninstalling testkube", err)

			ui.Verbose = originalVerbose

		},
	}

	cmd.Flags().StringVar(&name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "namespace from where to uninstall")

	return cmd
}
