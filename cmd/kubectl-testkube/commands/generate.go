package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/crds"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/generate"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "generate <resourceName>",
		Aliases:     []string{},
		Short:       "Generate resources commands",
		Annotations: map[string]string{cmdGroupAnnotation: cmdGroupCommands},
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		}}

	cmd.AddCommand(crds.NewCRDTestsCmd())
	cmd.AddCommand(generate.NewDocsCmd())

	return cmd
}
