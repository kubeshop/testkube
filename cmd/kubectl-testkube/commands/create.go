package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/agents"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testtriggers"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflowtemplates"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/webhooks"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/webhooktemplates"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/workflowtriggers"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateCmd() *cobra.Command {
	var crdOnly bool

	cmd := &cobra.Command{
		Use:     "create <resourceName>",
		Aliases: []string{"c"},
		Short:   "Create a Test Workflow, template, trigger, webhook or runner",
		Long: `Create one Test Workflow, Test Workflow Template, trigger, webhook or runner, from flags or a manifest.

Creating one that already exists fails; pass --update to update it instead (not for runners).
Webhooks and webhook templates are the exception: without --update, they ask whether to overwrite the existing one.

Boards, MCP Servers, LLM Providers, Skills and AI Agents are not created here: they are applied from YAML manifests with "testkube apply -f <file>", which creates or updates them.`,
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

	cmd.AddCommand(webhooks.NewCreateWebhookCmd())
	cmd.AddCommand(webhooktemplates.NewCreateWebhookTemplateCmd())
	cmd.AddCommand(workflowtriggers.NewCreateWorkflowTriggerCmd())
	cmd.AddCommand(testtriggers.NewCreateTestTriggerCmd())
	cmd.AddCommand(testworkflows.NewCreateTestWorkflowCmd())
	cmd.AddCommand(testworkflowtemplates.NewCreateTestWorkflowTemplateCmd())
	cmd.AddCommand(agents.NewCreateAgentCommand())

	cmd.PersistentFlags().BoolVar(&crdOnly, "crd-only", false, "generate only crd")

	return cmd
}
