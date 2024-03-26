package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/executors"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/templates"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsources"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflowtemplates"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/webhooks"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "delete <resourceName>",
		Aliases:     []string{"remove"},
		Short:       "Delete resources",
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

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct|cluster")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "should I show additional debug messages")

	cmd.AddCommand(tests.NewDeleteTestsCmd())
	cmd.AddCommand(testsuites.NewDeleteTestSuiteCmd())
	cmd.AddCommand(webhooks.NewDeleteWebhookCmd())
	cmd.AddCommand(executors.NewDeleteExecutorCmd())
	cmd.AddCommand(testsources.NewDeleteTestSourceCmd())
	cmd.AddCommand(templates.NewDeleteTemplateCmd())
	cmd.AddCommand(testworkflows.NewDeleteTestWorkflowCmd())
	cmd.AddCommand(testworkflowtemplates.NewDeleteTestWorkflowTemplateCmd())

	return cmd
}
