package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/marketplace"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewMarketplaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "marketplace",
		Aliases: []string{"mp"},
		Short:   "Browse and install TestWorkflows from the Testkube Marketplace",
		Long: `Browse, inspect, and install TestWorkflows from the Testkube Marketplace
(https://github.com/kubeshop/testkube-marketplace) without opening the dashboard.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
	}

	cmd.PersistentFlags().StringP("output", "o", "pretty", "output type: pretty|json|yaml|go")
	cmd.PersistentFlags().StringP("go-template", "", "{{.}}", "go template to render when --output=go")

	cmd.AddCommand(marketplace.NewListCmd())
	cmd.AddCommand(marketplace.NewCategoriesCmd())
	cmd.AddCommand(marketplace.NewGetCmd())
	cmd.AddCommand(marketplace.NewInstallCmd())

	return cmd
}
