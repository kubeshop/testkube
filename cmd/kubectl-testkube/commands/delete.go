package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/executors"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/webhooks"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <resourceName>",
		Aliases: []string{"remove"},
		Short:   "Delete resources",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			validator.PersistentPreRunVersionCheck(cmd, Version)
		}}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "testkube", "kubernetes namespace")

	cmd.AddCommand(tests.NewDeleteTestsCmd())
	cmd.AddCommand(testsuites.NewDeleteTestSuiteCmd())
	cmd.AddCommand(webhooks.NewDeleteWebhookCmd())
	cmd.AddCommand(executors.NewDeleteExecutorCmd())

	return cmd
}
