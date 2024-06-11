package debug

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDebugControlPlaneCmd creates a new cobra command to print the debug info to the CLI
func NewDebugControlPlaneCmd() *cobra.Command {
	var additionalLabels map[string]string
	var attachAgentLogs bool

	cmd := &cobra.Command{
		Use:     "controlplane",
		Aliases: []string{"ctl", "cp"},
		Short:   "Show debug info",
		Long:    "Get all the necessary information to debug an issue in Testkube Control Plane",
		Run: func(cmd *cobra.Command, args []string) {

			spinner := ui.NewSpinner("").WithWriter(os.Stderr)
			spinner, err := spinner.Start()
			ui.ExitOnError("starting spinner", err)

			namespace, err := cmd.Flags().GetString("namespace")
			ui.ExitOnError("getting namespace", err)

			ui.H1("Getting control plane logs")

			spinner.UpdateText("Getting Kubernetes pods")
			ui.H2("Kubernetes Pods in namespace:" + namespace)
			err = common.KubectlPrintPods(namespace)
			ui.WarnOnError("getting Kubernetes pods", err)

			spinner.UpdateText("Kubernetes Storage Classes")
			ui.H2("Kubernetes Storage Classes")
			err = common.KubectlPrintStorageClass(namespace)
			ui.WarnOnError("getting Kubernetes Storage Classes", err)

			spinner.UpdateText("API Server Logs")
			ui.H2("API Server Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-cloud-api"})
			ui.WarnOnError("getting api server logs", err)

			spinner.UpdateText("Worker Service Logs")
			ui.H2("Worker Service Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-worker-service"})
			ui.WarnOnError("getting worker service logs", err)

			spinner.UpdateText("UI Logs")
			ui.H2("UI Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-cloud-ui"})
			ui.WarnOnError("getting UI logs", err)

			spinner.UpdateText("UI Logs")
			ui.H2("Dex Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "dex"})
			ui.WarnOnError("getting Dex logs", err)

			spinner.UpdateText("UI Logs")
			ui.H2("Minio Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "minio"})
			ui.WarnOnError("getting MinIO logs", err)

			spinner.UpdateText("MongoDB logs")
			ui.H2("MongoDB logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "mongodb"})
			ui.WarnOnError("getting MongoDB logs", err)

			spinner.UpdateText("NATS Logs")
			ui.H2("NATS logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "nats"})
			ui.WarnOnError("getting worker service logs", err)

			spinner.UpdateText("Kubernetes Events")
			ui.H2("Kubernetes Events")
			err = common.KubectlPrintEvents(namespace)
			ui.WarnOnError("getting Kubernetes events", err)

			if cmd.Flag("attach-agent-log").Value.String() == "true" {
				spinner.UpdateText("UI Logs")
				ui.H2("Agent Logs")
				err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-agent"})
				ui.ExitOnError("getting agent logs", err)

				spinner.UpdateText("UI Logs")
				ui.H1("Agent debug info")
				client, _, err := common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				debug, err := GetDebugInfo(client)
				ui.ExitOnError("get debug info", err)

				PrintDebugInfo(debug)
			}

			spinner.Success("Testkube logs collected successfully")

		},
	}

	cmd.Flags().StringToStringVar(&additionalLabels, "labels", map[string]string{}, "Labels to filter logs by")
	cmd.Flags().BoolVar(&attachAgentLogs, "attach-agent-log", false, "Attach agent log to the output keep in mind to configure valid agent first in the Testkube CLI")

	return cmd

}
