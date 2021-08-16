package commands

import (
	"github.com/kubeshop/kubetest/pkg/process"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {

	RootCmd.AddCommand(installCmd)

	installCmd.Flags().String("chart", "./charts/kubetest", "chart name")
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Helm chart registry in current kubectl context",
	Long:  `Install can be configured with use of particular `,
	Run: func(cmd *cobra.Command, args []string) {

		chart := cmd.Flag("chart").Value.String()

		out, err := process.Execute("helm", "install", "kubetest", chart)
		ui.ExitOnError("executing helm install", err)

		ui.Info("Helm output", string(out))
	},
}
