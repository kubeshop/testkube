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
		PreRun: func(cmd *cobra.Command, args []string) {
			// Override parent's PersistentPreRun to skip version check
			// get context only displays local config and doesn't need network connectivity
		},
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			ui.NL()
			common.UiPrintContext(cfg)
		},
	}

	return cmd
}
