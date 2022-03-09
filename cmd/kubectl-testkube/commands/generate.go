package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/crds"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/generate"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate <resourceName>",
		Aliases: []string{},
		Short:   "Generate resources commands",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cmd.Help()
		},
		PersistentPreRun: validator.PersistentPreRunVersionCheckFunc(Version),
	}

	cmd.AddCommand(crds.NewCRDTestsCmd())
	cmd.AddCommand(generate.NewDocsCmd())

	return cmd
}
