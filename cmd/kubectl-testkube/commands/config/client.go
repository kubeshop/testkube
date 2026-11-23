package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewConfigureClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "client <proxy|direct|cluster>",
		Short:     "Set the client used to connect to the Testkube API",
		Long:      "Set the client used to connect to the Testkube API when --client isn't passed. Pass an empty string to go back to the default.",
		ValidArgs: config.ClientTypes,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please pass one of proxy, direct or cluster")
			}
			if args[0] == "" {
				return nil
			}
			return config.ValidateClientType(args[0])
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			cfg.Client = args[0]
			err = config.Save(cfg)
			ui.ExitOnError("saving config file", err)
			if cfg.Client == "" {
				ui.Success("Client reset to the default")
				return
			}
			ui.Success("New client set to", cfg.Client)
		},
	}

	return cmd
}
