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

func NewCreateCmd() *cobra.Command {
	var crdOnly bool

	cmd := &cobra.Command{
		Use:         "create <resourceName>",
		Aliases:     []string{"c"},
		Short:       "Create resource",
		Annotations: map[string]string{cmdGroupAnnotation: cmdGroupCommands},
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)
			common.UiContextHeader(cmd, cfg)

			if !crdOnly {
				validator.PersistentPreRunVersionCheck(cmd, common.Version)
			}
		}}

	cmd.AddCommand(tests.NewCreateTestsCmd())
	cmd.AddCommand(testsuites.NewCreateTestSuitesCmd())
	cmd.AddCommand(webhooks.NewCreateWebhookCmd())
	cmd.AddCommand(executors.NewCreateExecutorCmd())
	cmd.AddCommand(testsources.NewCreateTestSourceCmd())
	cmd.AddCommand(templates.NewCreateTemplateCmd())
	cmd.AddCommand(testworkflows.NewCreateTestWorkflowCmd())
	cmd.AddCommand(testworkflowtemplates.NewCreateTestWorkflowTemplateCmd())

	cmd.PersistentFlags().BoolVar(&crdOnly, "crd-only", false, "generate only crd")

	return cmd
}
