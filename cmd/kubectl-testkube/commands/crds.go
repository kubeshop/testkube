package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/crds"

	"github.com/spf13/cobra"
)

func NewCRDsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "crd",
		Aliases: []string{"crds"},
		Short:   "CRDs management commands",
		Long:    `CRD generation tools`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "testkube", "kubernetes namespace")

	cmd.AddCommand(crds.NewCRDTestsCmd())
	return cmd
}
