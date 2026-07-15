package testworkflows

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows/renderer"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	common2 "github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/ui/uicrd"
)

func NewGetTestWorkflowsCmd() *cobra.Command {
	var (
		selectors []string
		crdOnly   bool
		limit     int
	)

	cmd := &cobra.Command{
		Use:     "testworkflow [name]",
		Aliases: []string{"testworkflows", "tw"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Get all available test workflows",
		Long:    `Get all available test workflows. In cloud context (API key) the CLI fetches them from the connected Control Plane environment and ignores the namespace flag. In kubeconfig context it fetches them from the agent in the given namespace (default "testkube").`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if limit < 0 {
				return fmt.Errorf("--limit must not be negative")
			}
			return nil
		},

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
				fetchLimit := limit
				if limit > 0 {
					fetchLimit = limit + 1
				}
				workflows, err := client.ListTestWorkflowWithExecutions(strings.Join(selectors, ","), fetchLimit)
				ui.ExitOnError("getting all test workflows"+namespaceSuffix, err)

				if limit > 0 && len(workflows) == limit+1 {
					workflows = workflows[:limit]
					ui.NewStderrUI(false).Warn(fmt.Sprintf("Showing %d test workflows, more are available on the server. Use --limit 0 to fetch all.", limit))
				}

				if crdOnly {
					uicrd.PrintCRDs(common2.MapSlice(workflows, func(t testkube.TestWorkflowWithExecution) testworkflowsv1.TestWorkflow {
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
			ui.ExitOnError("getting test workflow"+namespaceSuffix, err)

			if crdOnly {
				uicrd.PrintCRD(testworkflows.MapTestWorkflowAPIToKube(*workflow.Workflow), "TestWorkflow", testworkflowsv1.GroupVersion)
			} else {
				err = render.Obj(cmd, *workflow.Workflow, os.Stdout, renderer.TestWorkflowRenderer)
				ui.ExitOnError("rendering obj", err)

				if workflow.LatestExecution != nil {
					ui.NL()
					err = render.Obj(cmd, *workflow.LatestExecution, os.Stdout, renderer.TestWorkflowExecutionRenderer)
					ui.ExitOnError("rendering obj", err)
					common.UIShellViewExecution(workflow.LatestExecution.Id)
				}
			}
		},
	}
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "render the fetched test workflows as crd yaml; does not read crds from the cluster")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of workflows to return, 0 to fetch all")

	return cmd
}
