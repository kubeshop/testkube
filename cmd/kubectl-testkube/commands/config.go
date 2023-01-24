package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/config"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/oauth"
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
	}

	cmd.AddCommand(config.NewConfigureNamespaceCmd())
	cmd.AddCommand(config.NewConfigureAPIURICmd())
	cmd.AddCommand(oauth.NewConfigureOAuthCmd())

	return cmd
}
