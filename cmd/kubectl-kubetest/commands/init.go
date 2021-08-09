package commands

import (
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {
	installCmd.Flags().String("namespace", "default", "namespace where kubetest should be installed to")

	RootCmd.AddCommand(installCmd)

	installCmd.Flags().String("port", ":8080", "kubetest api server port")
	RootCmd.AddCommand(connectCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Helm chart registry in current kubectl context",
	Long:  `Install can be configured with use of particular `,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Errf("NOT IMPLEMENTED")
	},
}

// TODO - remove after migrating API as API-server/delegator
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Creates new connection with use of port-forward to local machine",
	Long:  `temporary command - will be removed in incoming versions`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Errf("NOT IMPLEMENTED")
	},
}
