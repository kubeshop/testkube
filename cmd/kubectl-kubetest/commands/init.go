package commands

import (
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {

	RootCmd.AddCommand(installCmd)

	installCmd.Flags().String("namespace", "default", "namespace where kubetest should be installed to")
	installCmd.Flags().String("port", ":8080", "kubetest api server port")
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Helm chart registry in current kubectl context",
	Long:  `Install can be configured with use of particular `,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Errf("NOT IMPLEMENTED")
	},
}
