package commands

import (
	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {

}

func NewInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Helm chart registry in current kubectl context",
		Long:  `Install can be configured with use of particular `,
		Run: func(cmd *cobra.Command, args []string) {

			ui.Logo()

			chart := cmd.Flag("chart").Value.String()
			name := cmd.Flag("name").Value.String()
			namespace := cmd.Flag("namespace").Value.String()

			_, err := process.Execute("helm", "repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts")
			ui.ExitOnError("adding kubtest repo", err)

			_, err = process.Execute("helm", "repo", "update")
			ui.ExitOnError("updating helm repositories", err)

			out, err := process.Execute("helm", "install", "--namespace", namespace, name, chart)
			ui.ExitOnError("executing helm install", err)

			ui.Info("Helm output", string(out))
		},
	}

	cmd.Flags().String("chart", "kubeshop/kubtest", "chart name")
	cmd.Flags().String("name", "kubtest", "installation name")
	cmd.Flags().String("namespace", "default", "namespace where to install")
	return cmd
}
