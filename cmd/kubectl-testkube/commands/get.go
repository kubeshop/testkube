package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/artifacts"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/context"
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

func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "get <resourceName>",
		Aliases:     []string{"g"},
		Short:       "Get resources",
		Long:        `Get available resources, get single item or list`,
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

		},
	}

	cmd.AddCommand(tests.NewGetTestsCmd())
	cmd.AddCommand(testsuites.NewGetTestSuiteCmd())
	cmd.AddCommand(webhooks.NewGetWebhookCmd())
	cmd.AddCommand(executors.NewGetExecutorCmd())
	cmd.AddCommand(tests.NewGetExecutionCmd())
	cmd.AddCommand(artifacts.NewListArtifactsCmd())
	cmd.AddCommand(testsuites.NewTestSuiteExecutionCmd())
	cmd.AddCommand(testsources.NewGetTestSourceCmd())
	cmd.AddCommand(context.NewGetContextCmd())
	cmd.AddCommand(templates.NewGetTemplateCmd())
	cmd.AddCommand(testworkflows.NewGetTestWorkflowsCmd())
	cmd.AddCommand(testworkflows.NewGetTestWorkflowExecutionsCmd())
	cmd.AddCommand(testworkflowtemplates.NewGetTestWorkflowTemplatesCmd())

	cmd.PersistentFlags().StringP("output", "o", "pretty", "output type can be one of json|yaml|pretty|go")
	cmd.PersistentFlags().StringP("go-template", "", "{{.}}", "go template to render")

	return cmd
}
