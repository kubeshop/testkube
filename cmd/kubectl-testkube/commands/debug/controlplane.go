package debug

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewShowDebugInfoCmd creates a new cobra command to print the debug info to the CLI
func NewDebugControlPlaneCmd() *cobra.Command {
	var additionalLabels map[string]string

	cmd := &cobra.Command{
		Use:     "controlplane",
		Aliases: []string{"ctl", "cp"},
		Short:   "Show debug info",
		Long:    "Get all the necessary information to debug an issue in Testkube Control Plane",
		Run: func(cmd *cobra.Command, args []string) {
			ui.H1("Getting control plane logs")

			namespace, err := cmd.Flags().GetString("namespace")
			ui.ExitOnError("getting namespace", err)

			ui.H2("API Server Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-cloud-api"})
			ui.WarnOnError("getting api server logs", err)

			ui.H2("Worker Service Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-worker-service"})
			ui.WarnOnError("getting worker service logs", err)

			ui.H2("UI Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "testkube-cloud-ui"})
			ui.WarnOnError("getting UI logs", err)

			ui.H2("Dex Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "dex"})
			ui.WarnOnError("getting Dex logs", err)

			ui.H2("Minio Logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "minio"})
			ui.WarnOnError("getting MinIO logs", err)

			ui.H2("MongoDB logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "mongodb"})
			ui.WarnOnError("getting MongoDB logs", err)

			ui.H2("NATS logs")
			err = common.KubectlPrintLogs(namespace, map[string]string{"app.kubernetes.io/name": "nats"})
			ui.WarnOnError("getting worker service logs", err)

			ui.H2("Agent debug info")
			client, _, err := common.GetClient(cmd)
			ui.WarnOnError("getting client", err)

			debug, err := GetDebugInfo(client)
			ui.WarnOnError("get debug info", err)

			PrintDebugInfo(debug)
		},
	}

	cmd.Flags().StringToStringVar(&additionalLabels, "labels", map[string]string{}, "Labels to filter logs by")

	return cmd

}
