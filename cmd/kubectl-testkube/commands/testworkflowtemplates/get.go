package testworkflowtemplates

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflowtemplates/renderer"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/ui/uicrd"
)

func NewGetTestWorkflowTemplatesCmd() *cobra.Command {
	var (
		selectors []string
		crdOnly   bool
	)

	cmd := &cobra.Command{
		Use:     "testworkflowtemplate [name]",
		Aliases: []string{"testworkflowtemplates", "twt"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Get all available test workflow templates",
		Long:    `Get all available test workflow templates. In cloud context (API key) the CLI fetches them from the connected Control Plane environment and ignores the namespace flag. In kubeconfig context it fetches them from the agent in the given namespace (default "testkube").`,

		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			// Namespace only scopes the query in kubeconfig context, so keep it out of cloud-context error messages.
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)
			namespaceSuffix := " in namespace " + namespace
			if cfg.ContextType == config.ContextTypeCloud {
				namespaceSuffix = ""
			}

			if len(args) == 0 {
				templates, err := client.ListTestWorkflowTemplates(strings.Join(selectors, ","))
				ui.ExitOnError("getting all test workflow templates"+namespaceSuffix, err)

				if crdOnly {
					uicrd.PrintCRDs(testworkflows.MapTemplateListAPIToKube(templates).Items, "TestWorkflowTemplate", testworkflowsv1.GroupVersion)
				} else {
					err = render.List(cmd, templates, os.Stdout)
					ui.PrintOnError("Rendering list", err)
				}
				return
			}

			name := args[0]
			template, err := client.GetTestWorkflowTemplate(name)
			ui.ExitOnError("getting test workflow template"+namespaceSuffix, err)

			if crdOnly {
				uicrd.PrintCRD(testworkflows.MapTestWorkflowTemplateAPIToKube(template), "TestWorkflowTemplate", testworkflowsv1.GroupVersion)
			} else {
				err = render.Obj(cmd, template, os.Stdout, renderer.TestWorkflowTemplateRenderer)
				ui.ExitOnError("rendering obj", err)
			}
		},
	}
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "render the fetched test workflow templates as crd yaml; does not read crds from the cluster")

	return cmd
}
