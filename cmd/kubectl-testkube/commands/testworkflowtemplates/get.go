package testworkflowtemplates

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflowtemplates/renderer"
	"github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
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
		Long:    `Getting all available test workflow templates from given namespace - if no namespace given "testkube" namespace is used`,

		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) == 0 {
				templates, err := client.ListTestWorkflowTemplates(strings.Join(selectors, ","))
				ui.ExitOnError("getting all test workflow templates in namespace "+namespace, err)

				if crdOnly {
					ui.PrintCRDs(testworkflows.MapTemplateListAPIToKube(templates).Items, "TestWorkflowTemplate", testworkflowsv1.GroupVersion)
				} else {
					err = render.List(cmd, templates, os.Stdout)
					ui.PrintOnError("Rendering list", err)
				}
				return
			}

			name := args[0]
			template, err := client.GetTestWorkflowTemplate(name)
			ui.ExitOnError("getting test workflow in namespace "+namespace, err)

			if crdOnly {
				ui.PrintCRD(testworkflows.MapTestWorkflowTemplateAPIToKube(template), "TestWorkflowTemplate", testworkflowsv1.GroupVersion)
			} else {
				err = render.Obj(cmd, template, os.Stdout, renderer.TestWorkflowTemplateRenderer)
				ui.ExitOnError("rendering obj", err)
			}
		},
	}
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test workflow template crd")

	return cmd
}
