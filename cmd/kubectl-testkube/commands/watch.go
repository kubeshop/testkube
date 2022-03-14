package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "watch <resourceName>",
		Aliases: []string{"r", "start"},
		Short:   "Watch tests or test suites",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			validator.PersistentPreRunVersionCheck(cmd, Version)
		}}

	cmd.AddCommand(tests.NewWatchExecutionCmd())
	cmd.AddCommand(testsuites.NewWatchTestSuiteExecutionCmd())

	return cmd
}
