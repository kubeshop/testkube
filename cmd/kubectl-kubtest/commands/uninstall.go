package commands

import (
	"strings"

	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUninstallCmd() *cobra.Command {
	var name, namespace string
	var removeCRDs bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Helm chart registry in current kubectl context",
		Long:  `Uninstall Helm chart registry in current kubectl context`,
		Run: func(cmd *cobra.Command, args []string) {

			ui.Verbose = true
			ui.Logo()

			_, err := process.Execute("helm", "uninstall", "--namespace", namespace, name)
			ui.ExitOnError("uninstalling kubtest", err)

			if removeCRDs {
				_, err = process.Execute("kubectl", "delete", "crds", "--namespace", namespace, "scripts.tests.kubtest.io", "executors.executor.kubtest.io")
				ui.ExitOnError("uninstalling CRDs", err)
			}

			if isIngressInstalled() {
				_, err := process.Execute("helm", "uninstall", "--namespace", namespace, IngressControllerName)
				ui.ExitOnError("uninstalling ingress controller", err)
			}
		},
	}

	cmd.Flags().StringVar(&name, "name", "kubtest", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "namespace where to install")
	cmd.Flags().BoolVar(&removeCRDs, "remove-crds", false, "wipe out Executors and Scripts CRDs")

	return cmd
}

func isIngressInstalled() bool {
	output, _ := process.Execute("helm", "list", "--namespace", namespace)
	return strings.Contains(string(output), IngressControllerName)
}
