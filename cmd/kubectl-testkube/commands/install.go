package commands

import (
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
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

			k8sClient, err := k8sclient.ConnectToK8s()
			if err != nil {
				ui.Info("Cannot connect to cluster", err.Error())
				return
			}

			if installIngress {
				err = installIngressController(k8sClient, namespace)
				ui.PrintOnError("installing ingress controller", err)
			}

			_, err = process.Execute("helm", "repo", "add", "kubeshop", "https://kubeshop.github.io/helm-charts")
			ui.WarnOnError("adding testkube repo", err)

			_, err = process.Execute("helm", "repo", "update")
			ui.ExitOnError("updating helm repositories", err)

			out, err := process.Execute("helm", "upgrade", "--install", "--create-namespace", "--namespace", namespace, name, chart)
			ui.ExitOnError("executing helm install", err)
			ui.Info("Helm install output", string(out))

			err = printDashboardAddress(k8sClient, namespace)
			if installIngress {
				ui.ExitOnError("getting dashboard address", err)
			}
		},
	}

	cmd.Flags().StringVar(&chart, "chart", "kubeshop/testkube", "chart name")
	cmd.Flags().StringVar(&name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "namespace where to install")
	cmd.Flags().BoolVarP(&installIngress, "ingress", "i", false, "install ingress if not present in the cluster to expose the endpoint for the dashboard")
	return cmd
}

func installIngressController(k8sClient *kubernetes.Clientset, namespace string) error {
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

	err = k8sclient.WaitForPodsReady(k8sClient, namespace, IngressControllerName, 50*time.Second)
	if err != nil {
		return err
	}
	return nil
}

func printDashboardAddress(k8sClient *kubernetes.Clientset, namespace string) error {
	//TODO: get automatically the name of the api-server
	ingressAddress, err := k8sclient.GetIngressAddress(k8sClient, IngressApiServerName, namespace)
	if err != nil {
		return fmt.Errorf("cannot get the ingress address %w", err)
	}

	completeServerApiAddress := fmt.Sprintf("%s%s/%s/%s/executions/", DashboardURI, ingressAddress, DashboardPrefix, CurrentApiVersion)

	ui.Info("testkube dashboard can be accessed at the address ", completeServerApiAddress)
	ui.Info("a certificate should be added to the ingress to make connection secure")
	return nil
}
