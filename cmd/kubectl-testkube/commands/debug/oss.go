package debug

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDebugOssCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "oss",
		Aliases: []string{"o"},
		Short:   "Show OSS installation debug info",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if cfg.ContextType != config.ContextTypeKubeconfig {
				ui.Errf("OSS debug is only available for kubeconfig context, use `testkube set context` to set kubeconfig context, or `testkube debug agent|controlplane` to debug other variants of Testkube")
				return
			}

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			d, err := GetDebugInfo(client)
			ui.ExitOnError("get debug info", err)

			PrintDebugInfo(d)
		},
	}

	return cmd
}
