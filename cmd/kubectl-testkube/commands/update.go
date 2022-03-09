package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <resourceName>",
		Aliases: []string{"u"},
		Short:   "Update resource",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cmd.Help()
		},
		PersistentPreRun: validator.PersistentPreRunVersionCheckFunc(Version),
	}

	cmd.AddCommand(tests.NewUpdateTestsCmd())
	cmd.AddCommand(testsuites.NewUpdateTestSuitesCmd())

	return cmd
}
