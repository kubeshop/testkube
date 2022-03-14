package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewAbortCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "abort <resourceName>",
		Short: "Abort tests or test suites",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			validator.PersistentPreRunVersionCheck(cmd, Version)
		}}

	cmd.AddCommand(tests.NewAbortExecutionCmd())

	return cmd
}
