package commands

import (
	"fmt"
	"time"

	"github.com/kubeshop/kubtest/pkg/k8sclient"
	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {

}

func NewInstallCmd() *cobra.Command {
	var chart, name, namespace string
	installIngress := false
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Helm chart registry in current kubectl context",
		Long:  `Install can be configured with use of particular `,
		Run: func(cmd *cobra.Command, args []string) {

			ui.Verbose = true

			ui.Logo()

			var err error
			if installIngress {
				err = installIngressController(namespace)
				ui.PrintOnError("installing ingress controller", err)
				time.Sleep(20 * time.Second)
			}

			_, err = process.Execute("helm", "repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts")
			ui.ExitOnError("adding kubtest repo", err)

			_, err = process.Execute("helm", "repo", "update")
			ui.ExitOnError("updating helm repositories", err)

			out, err := process.Execute("helm", "install", "--namespace", namespace, name, chart)
			ui.ExitOnError("executing helm install", err)

			ui.Info("Helm install output", string(out))

			k8sClient, err := k8sclient.ConnectToK8s()
			if err != nil {
				ui.Info("Cannot get the ingress info", err.Error())
				return
			}

			//TODO: get automatically the name of the api-server
			ingressAddress, err := k8sclient.GetIngressAddress(k8sClient, IngressApiServerName, namespace)
			if err != nil {
				ui.Info("Cannot get the ingress address", err.Error())
				return
			}

			// TODO: Add version from constant
			completeServerApiAddress := fmt.Sprintf("%s%s/%s/v1/executions/", DashboardURI, ingressAddress, DashboardPrefix)

			ui.Info("Kubtest dashboard can be accessed at the address ", completeServerApiAddress)

		},
	}

	cmd.Flags().StringVar(&chart, "chart", "kubeshop/kubtest", "chart name")
	cmd.Flags().StringVar(&name, "name", "kubtest", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "namespace where to install")
	cmd.Flags().BoolVarP(&installIngress, "ingress", "i", false, "install ingress if not present in the cluster to expose the endpoint for the dashboard")
	return cmd
}

func installIngressController(namespace string) error {
	_, err := process.Execute("helm", "repo", "add", "ingress-nginx", "https://kubernetes.github.io/ingress-nginx")
	if err != nil {
		return err
	}

	_, err = process.Execute("helm", "repo", "update")
	if err != nil {
		return err
	}

	_, err = process.Execute("helm", "install", "--namespace", namespace, IngressControllerName, "ingress-nginx/ingress-nginx")
	if err != nil {
		return err
	}

	return nil
}
