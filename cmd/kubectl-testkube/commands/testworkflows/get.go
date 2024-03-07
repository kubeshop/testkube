package testworkflows

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows/renderer"
	common2 "github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/tcl/mapperstcl/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetTestWorkflowsCmd() *cobra.Command {
	var (
		selectors []string
		crdOnly   bool
	)

	cmd := &cobra.Command{
		Use:     "testworkflow [name]",
		Aliases: []string{"testworkflows", "tw"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Get all available test workflows",
		Long:    `Getting all available test workflows from given namespace - if no namespace given "testkube" namespace is used`,

		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) == 0 {
				workflows, err := client.ListTestWorkflowWithExecutions(strings.Join(selectors, ","))
				ui.ExitOnError("getting all test workflows in namespace "+namespace, err)

				if crdOnly {
					ui.PrintCRDs(common2.MapSlice(workflows, func(t testkube.TestWorkflowWithExecution) testworkflowsv1.TestWorkflow {
						return *testworkflows.MapAPIToKube(t.Workflow)
					}), "TestWorkflow", testworkflowsv1.GroupVersion)
				} else {
					err = render.List(cmd, workflows, os.Stdout)
					ui.PrintOnError("Rendering list", err)
				}
				return
			}

			name := args[0]
			workflow, err := client.GetTestWorkflowWithExecution(name)
			ui.ExitOnError("getting test workflow in namespace "+namespace, err)

			if crdOnly {
				ui.PrintCRD(testworkflows.MapTestWorkflowAPIToKube(*workflow.Workflow), "TestWorkflow", testworkflowsv1.GroupVersion)
			} else {
				err = render.Obj(cmd, *workflow.Workflow, os.Stdout, renderer.TestWorkflowRenderer)
				ui.ExitOnError("rendering obj", err)

				if workflow.LatestExecution != nil {
					ui.NL()
					err = render.Obj(cmd, *workflow.LatestExecution, os.Stdout, renderer.TestWorkflowExecutionRenderer)
					ui.ExitOnError("rendering obj", err)
				}
			}
		},
	}
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test workflow crd")

	return cmd
}
