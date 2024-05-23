package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	commands "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/config"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/oauth"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config <feature> <value>",
		Aliases: []string{"set", "configure"},
		Short:   "Set feature configuration value",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)
			common.UiContextHeader(cmd, cfg)
		},
	}

	cmd.AddCommand(commands.NewConfigureNamespaceCmd())
	cmd.AddCommand(commands.NewConfigureAPIURICmd())
	cmd.AddCommand(commands.NewConfigureHeadersCmd())
	cmd.AddCommand(oauth.NewConfigureOAuthCmd())
	cmd.AddCommand(commands.NewConfigureAPIServerNameCmd())
	cmd.AddCommand(commands.NewConfigureAPIServerPortCmd())

	return cmd
}
