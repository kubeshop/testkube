package debug

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDebugControlPlaneCmd creates a new cobra command to print the debug info to the CLI
func NewDebugControlPlaneCmd() *cobra.Command {
	const defaultCPNamespace = "testkube-enterprise"
	var features common.CommaList

	cmd := &cobra.Command{
		Use:     "controlplane",
		Aliases: []string{"ctl", "cp"},
		Short:   "Show debug info",
		Long:    "Get all the necessary information to debug an issue in Testkube Control Plane",
		Run: func(cmd *cobra.Command, args []string) {
			namespace := common.UiGetNamespace(cmd, defaultCPNamespace)

			ui.H1("Getting Control Plane insights, namespace: " + namespace)

			if features.Enabled("pods") {
				ui.H2("Pods")
				err := common.KubectlPrintPods(namespace)
				ui.WarnOnError("getting Kubernetes pods", err)

				ui.NL(3)
				err = common.KubectlDescribePods(namespace)
				ui.WarnOnError("describing Kubernetes pods", err)
			}

			if features.Enabled("servives") {
				ui.H2("Services")
				err := common.KubectlGetServices(namespace)
				ui.WarnOnError("describing Kubernetes pods", err)

				ui.NL(3)
				err = common.KubectlDescribeServices(namespace)
				ui.WarnOnError("describing Kubernetes services", err)
			}

			if features.Enabled("ingresses") {
				ui.H2("Ingresses")
				err := common.KubectlGetIngresses(namespace)
				ui.WarnOnError("describing Kubernetes ingresses", err)

				ui.NL(3)
				err = common.KubectlDescribeIngresses(namespace)
				ui.WarnOnError("describing Kubernetes services", err)
			}

			if features.Enabled("storageclasses") {
				ui.H2("Storage Classes")
				err := common.KubectlGetStorageClass(namespace)
				ui.WarnOnError("getting Kubernetes Storage Classes", err)
			}

			if features.Enabled("api") {
				ui.H2("API Server Logs")
				err := common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-cloud-api"})
				ui.WarnOnError("getting api server logs", err)
			}

			if features.Enabled("worker") {
				ui.H2("Worker Service Logs")
				err := common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-worker-service"})
				ui.WarnOnError("getting worker service logs", err)
			}

			if features.Enabled("ui") {
				ui.H2("UI Logs")
				err := common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-cloud-ui"})
				ui.WarnOnError("getting UI logs", err)
			}

			if features.Enabled("dex") {
				ui.H2("Dex Logs")
				err := common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "dex"})
				ui.WarnOnError("getting Dex logs", err)
			}

			if features.Enabled("minio") {
				ui.H2("Minio Logs")
				err := common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "minio"})
				ui.WarnOnError("getting MinIO logs", err)
			}

			if features.Enabled("mongo") {
				ui.H2("MongoDB logs")
				err := common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "mongodb"})
				ui.WarnOnError("getting MongoDB logs", err)
			}

			if features.Enabled("nats") {
				ui.H2("NATS logs")
				err := common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "nats"})
				ui.WarnOnError("getting worker service logs", err)
			}

			if features.Enabled("events") {
				ui.H2("Kubernetes Events")
				err := common.KubectlPrintEvents(namespace)
				ui.WarnOnError("getting Kubernetes events", err)
			}

		},
	}

	cmd.Flags().Var(&features, "show", "Comma-separated list of features to show, one of: pods,services,storageclasses,api,worker,ui,dex,nats,mongo,minio, defaults to all")

	return cmd
}
