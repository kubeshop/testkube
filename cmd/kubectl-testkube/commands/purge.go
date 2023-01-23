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
		Short:   "Uninstall Helm chart registry from current kubectl context",
		Long:    `Uninstall Helm chart registry from current kubectl context`,
		Aliases: []string{"uninstall"},
		Run: func(cmd *cobra.Command, args []string) {

			ui.Verbose = true

			_, err := process.Execute("helm", "uninstall", "--namespace", namespace, name)
			ui.PrintOnError("uninstalling testkube", err)
		},
	}

	cmd.Flags().StringVar(&name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "namespace from where to uninstall")

	return cmd
}
