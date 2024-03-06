package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "run <resourceName>",
		Aliases:     []string{"r", "start"},
		Short:       "Runs tests, test suites or test workflows",
		Annotations: map[string]string{cmdGroupAnnotation: cmdGroupCommands},
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)
			common.UiContextHeader(cmd, cfg)

			validator.PersistentPreRunVersionCheck(cmd, common.Version)
		}}

	cmd.AddCommand(tests.NewRunTestCmd())
	cmd.AddCommand(testsuites.NewRunTestSuiteCmd())
	cmd.AddCommand(testworkflows.NewRunTestWorkflowCmd())

	return cmd
}
