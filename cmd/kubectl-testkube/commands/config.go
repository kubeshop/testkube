package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	commands "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/config"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewConfigCmd() *cobra.Command {
	var list bool

	cmd := &cobra.Command{
		Use:     "config <feature> <value>",
		Aliases: []string{"set", "configure"},
		Short:   "Set feature configuration value",
		Run: func(cmd *cobra.Command, args []string) {
			if list {
				cfg, err := config.Load()
				ui.ExitOnError("loading config file", err)

				ui.NL()
				ui.Properties([][]string{
					{"Context type     ", string(cfg.ContextType)},
					{"Namespace        ", cfg.Namespace},
					{"API URI          ", cfg.APIURI},
					{"API Server Name  ", cfg.APIServerName},
					{"API Server Port  ", fmt.Sprintf("%d", cfg.APIServerPort)},
					{"Headers          ", testkube.MapToString(cfg.Headers)},
					{"Telemetry Enabled", fmt.Sprintf("%t", cfg.TelemetryEnabled)},
				})
				return
			}

			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)
			common.UiContextHeader(cmd, cfg)
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "list current configuration values")

	cmd.AddCommand(commands.NewConfigureNamespaceCmd())
	cmd.AddCommand(commands.NewConfigureAPIURICmd())
	cmd.AddCommand(commands.NewConfigureHeadersCmd())
	cmd.AddCommand(commands.NewConfigureAPIServerNameCmd())
	cmd.AddCommand(commands.NewConfigureAPIServerPortCmd())

	return cmd
}
