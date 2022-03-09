package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run <resourceName>",
		Aliases: []string{"r", "start"},
		Short:   "Runs tests or test suites",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cmd.Help()
		},
		PersistentPreRun: validator.PersistentPreRunVersionCheckFunc(Version),
	}

	cmd.AddCommand(tests.NewRunTestCmd())
	cmd.AddCommand(testsuites.NewRunTestSuiteCmd())

	return cmd
}
