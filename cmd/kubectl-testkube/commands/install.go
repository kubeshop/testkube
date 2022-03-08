package commands

import (
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewInstallCmd() *cobra.Command {
	var (
		noDashboard            bool
		noMinio                bool
		noJetstack             bool
		chart, name, namespace string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Helm chart registry in current kubectl context and update dependencies",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			err := HelmUpgradeOrInstalTestkube(name, namespace, chart, noDashboard, noMinio, noJetstack)
			ui.ExitOnError("Installing testkube", err)
		},
	}

	cmd.Flags().StringVar(&chart, "chart", "kubeshop/testkube", "chart name")
	cmd.Flags().StringVar(&name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "namespace where to install")

	cmd.Flags().BoolVar(&noMinio, "no-minio", false, "don't install MinIO")
	cmd.Flags().BoolVar(&noDashboard, "no-dashboard", false, "don't install dashboard")
	cmd.Flags().BoolVar(&noJetstack, "no-jetstack", false, "don't install Jetstack")

	return cmd
}
