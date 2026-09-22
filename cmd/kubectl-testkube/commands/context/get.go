package context

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetContextCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "context <value>",
		Short: "Set context for Testkube Pro",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					common.ConfigFileHint,
					err,
				))
			}

			ui.NL()
			common.UiPrintContext(cfg)
		},
	}

	return cmd
}
