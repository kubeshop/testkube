package commands

import (
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUninstallCmd() *cobra.Command {
	var name, namespace string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Helm chart registry in current kubectl context",
		Long:  `Uninstall Helm chart registry in current kubectl context`,
		Run: func(cmd *cobra.Command, args []string) {

			ui.Verbose = true

			_, err := process.Execute("helm", "uninstall", "--namespace", namespace, name)
			ui.PrintOnError("uninstalling testkube", err)
		},
	}

	cmd.Flags().StringVar(&name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "namespace where to install")

	return cmd
}
