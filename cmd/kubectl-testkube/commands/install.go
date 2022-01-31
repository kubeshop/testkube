package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	noDashboard bool
	noMinio     bool
	noJetstack  bool
)

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
			if err != nil && !strings.Contains(err.Error(), "Error: repository name (kubeshop) already exists, please specify a different name") {
				ui.WarnOnError("adding testkube repo", err)
			}

			_, err = process.Execute(helmPath, "repo", "update")
			ui.ExitOnError("updating helm repositories", err)

			command := []string{"upgrade", "--install", "--create-namespace", "--namespace", namespace}
			command = append(command, "--set", fmt.Sprintf("api-server.minio.enabled=%t", !noMinio))
			command = append(command, "--set", fmt.Sprintf("testkube-dashboard.enabled=%t", !noDashboard))
			command = append(command, "--set", fmt.Sprintf("api-server.jetstack.enabled=%t", !noJetstack))
			command = append(command, name, chart)

			out, err := process.Execute(helmPath, command...)

			ui.ExitOnError("executing helm install", err)
			ui.Info("Helm install output", string(out))
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
