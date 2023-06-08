package config

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewConfigureAPIServerPortCmd is api server port config command
func NewConfigureAPIServerPortCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-server-port <value>",
		Short: "Set api server port for testkube client",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please pass valid api server port value")
			}

			if _, err := strconv.Atoi(args[0]); err != nil {
				return fmt.Errorf("please pass integer api server port value: %w", err)
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			port, err := strconv.Atoi(args[0])
			ui.ExitOnError("converting port value", err)

			cfg.APIServerPort = port
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			ui.Success("New api server port set to", strconv.Itoa(cfg.APIServerPort))
		},
	}

	return cmd
}
