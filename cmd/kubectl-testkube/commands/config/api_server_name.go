package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewConfigureAPIServerNameCmd is api server name config command
func NewConfigureAPIServerNameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-server-name <value>",
		Short: "Set api server name for testkube client",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please pass valid api server name value")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.APIServerName = args[0]
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New api server name set to", cfg.APIServerName)
		},
	}

	return cmd
}
