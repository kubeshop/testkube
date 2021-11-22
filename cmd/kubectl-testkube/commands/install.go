package commands

import (
	"os/exec"

	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {

}

func NewInstallCmd() *cobra.Command {
	var chart, name, namespace string
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install Helm chart registry in current kubectl context",
		Long:    `Install can be configured with use of particular `,
		Aliases: []string{"update", "upgrade"},
		Run: func(cmd *cobra.Command, args []string) {

			ui.Verbose = true

			ui.Logo()
			var err error

			helmPath, err := exec.LookPath("helm")
			ui.ExitOnError("checking helm installation path", err)

			_, err = process.Execute(helmPath, "repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts")
			ui.WarnOnError("adding testkube repo", err)

			_, err = process.Execute(helmPath, "repo", "update")
			ui.ExitOnError("updating helm repositories", err)

			out, err := process.Execute(helmPath, "upgrade", "--install", "--create-namespace", "--namespace", namespace, name, chart)
			ui.ExitOnError("executing helm install", err)
			ui.Info("Helm install output", string(out))

		},
	}

	cmd.Flags().StringVar(&chart, "chart", "kubeshop/testkube", "chart name")
	cmd.Flags().StringVar(&name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "namespace where to install")
	return cmd
}
