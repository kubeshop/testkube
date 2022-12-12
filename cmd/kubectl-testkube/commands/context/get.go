package context

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetContextCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "context <value>",
		Short: "Set namespace for testkube client",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			ui.Info("Your Testkube Cloud Context")
			ui.NL()
			uiPrintCloudContext(cfg.CloudContext)
		},
	}

	return cmd
}
