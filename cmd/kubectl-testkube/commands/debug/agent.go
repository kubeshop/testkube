package debug

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

const defaultAgentNamespace = "testkube"

func NewDebugAgentCmd() *cobra.Command {
	var show common.CommaList

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Debug Agent info",
		Run:   RunDebugAgentCmdFunc(&show),
	}

	cmd.Flags().Var(&show, "show", "Comma-separated list of features to show, one of: pods,services,storageclasses,api,worker,ui,dex,nats,mongo,minio, defaults to all")

	return cmd
}

func RunDebugAgentCmdFunc(features *common.CommaList) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		ui.ExitOnError("loading config file", err)
		ui.NL()

		ui.H1("Agent Insights")

		if cfg.ContextType != config.ContextTypeCloud {
			ui.Errf("Agent debug is only available for cloud context")
			ui.NL()
			ui.ShellCommand("Please try command below to set your context into Cloud mode", `testkube set context -o <org> -e <env> -k <api-key> `)
			ui.NL()
			return
		}

		namespace := common.UiGetNamespace(cmd, defaultAgentNamespace)

		if features.Enabled("pods") {
			ui.H2("Pods")
			err = common.KubectlPrintPods(namespace)
			ui.WarnOnError("getting Kubernetes pods", err)

			ui.NL(3)
			err = common.KubectlDescribePods(namespace)
			ui.WarnOnError("describing Kubernetes pods", err)
		}

		if features.Enabled("servives") {
			ui.H2("Services")
			err = common.KubectlGetServices(namespace)
			ui.WarnOnError("describing Kubernetes pods", err)

			ui.NL(3)
			err = common.KubectlDescribeServices(namespace)
			ui.WarnOnError("describing Kubernetes services", err)
		}

		if features.Enabled("ingresses") {
			ui.H2("Ingresses")
			err = common.KubectlGetIngresses(namespace)
			ui.WarnOnError("describing Kubernetes ingresses", err)
		}

		if features.Enabled("agent") {
			ui.H2("Agent API Logs")
			err = common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "api-server"})
			ui.ExitOnError("getting agent logs", err)
			ui.NL(2)
		}

		if features.Enabled("nats") {
			ui.H2("NATS logs")
			err = common.KubectlLogs(namespace, map[string]string{"app.kubernetes.io/name": "nats"})
			ui.WarnOnError("getting worker service logs", err)
		}

		if features.Enabled("events") {
			ui.H2("Kubernetes Events")
			err = common.KubectlPrintEvents(namespace)
			ui.WarnOnError("getting Kubernetes events", err)
		}

		client, _, err := common.GetClient(cmd)
		ui.ExitOnError("getting client", err)

		if features.Enabled("debug") {
			ui.H2("Agent connection")

			debug, err := GetDebugInfo(client)
			ui.ExitOnError("connecting to Control Plane", err)
			PrintDebugInfo(debug)
			ui.NL(2)

			common.UiPrintContext(cfg)
		}

		if features.Enabled("connection") {
			i, err := client.GetServerInfo()
			if err != nil {
				ui.Errf("Error %v", err)
				ui.NL()
				ui.Info("Possible reasons:")
				ui.Warn("- Please check if your agent organization and environment are set correctly")
				ui.Warn("- Please check if your API token is set correctly")
				ui.NL()
			} else {
				ui.Warn("Agent correctly connected to cloud:\n")
				ui.InfoGrid(map[string]string{
					"Agent version  ": i.Version,
					"Agent namespace": i.Namespace,
				})
			}
		}
		ui.NL()
	}
}
