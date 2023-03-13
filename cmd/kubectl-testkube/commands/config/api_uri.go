package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewConfigureAPIURICmd is api uri config command
func NewConfigureAPIURICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-uri <value>",
		Short: "Set api uri for testkube client",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please pass valid api uri value")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.APIURI = args[0]
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New api uri set to", cfg.APIURI)
		},
	}

	return cmd
}
