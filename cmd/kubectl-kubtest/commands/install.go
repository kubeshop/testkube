package commands

import (
	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {

}

func NewInstallCmd() *cobra.Command {
	var chart, name, namespace string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Helm chart registry in current kubectl context",
		Long:  `Install can be configured with use of particular `,
		Run: func(cmd *cobra.Command, args []string) {

			ui.Verbose = true

			ui.Logo()

			_, err := process.Execute("helm", "repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts")
			ui.WarnOnError("adding kubtest repo", err)

			_, err = process.Execute("helm", "repo", "update")
			ui.ExitOnError("updating helm repositories", err)

			out, err := process.Execute("helm", "upgrade", "--install", "--namespace", namespace, name, chart)
			ui.ExitOnError("executing helm install", err)

			ui.Info("Helm output", string(out))
		},
	}

	cmd.Flags().StringVar(&chart, "chart", "kubeshop/kubtest", "chart name")
	cmd.Flags().StringVar(&name, "name", "kubtest", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "namespace where to install")

	return cmd
}
