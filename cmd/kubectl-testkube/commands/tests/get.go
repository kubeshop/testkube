package tests

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests/renderer"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetTestsCmd() *cobra.Command {
	var (
		selectors   []string
		noExecution bool
		crdOnly     bool
	)

	cmd := &cobra.Command{
		Use:     "test <testName>",
		Aliases: []string{"tests", "t"},
		Short:   "Get all available tests",
		Long:    `Getting all available tests from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			var name string
			firstEntry := true
			if len(args) > 0 {
				name = args[0]
				test, err := client.GetTestWithExecution(name)
				ui.ExitOnError("getting test in namespace "+namespace, err)

				if test.Test != nil {
					if crdOnly {
						test.Test.QuoteTestTextFields()
						common.UIPrintCRD(crd.TemplateTest, test.Test, &firstEntry)
						return
					}

					ui.NL()
					ui.Info("Test:")
					err = render.Obj(cmd, *test.Test, os.Stdout, renderer.TestRenderer)
					ui.ExitOnError("rendering obj", err)
				}

				if test.LatestExecution != nil && !noExecution {
					ui.NL()
					ui.Info("Latest execution")
					err = render.Obj(cmd, *test.LatestExecution, os.Stdout, renderer.ExecutionRenderer)
					ui.ExitOnError("rendering obj", err)
				}

			} else {
				if noExecution {
					tests, err := client.ListTests(strings.Join(selectors, ","))
					ui.ExitOnError("getting all tests in namespace "+namespace, err)

					if crdOnly {
						for _, test := range tests {
							test.QuoteTestTextFields()
							common.UIPrintCRD(crd.TemplateTest, test, &firstEntry)
						}

						return
					}

					err = render.List(cmd, tests, os.Stdout)
					ui.PrintOnError("Rendering list", err)
				} else {
					tests, err := client.ListTestWithExecutionSummaries(strings.Join(selectors, ","))
					ui.ExitOnError("getting all test with execution summaries in namespace "+namespace, err)
					if crdOnly {
						for _, test := range tests {
							if test.Test != nil {
								test.Test.QuoteTestTextFields()
								common.UIPrintCRD(crd.TemplateTest, test.Test, &firstEntry)
							}
						}

						return
					}

					err = render.List(cmd, tests, os.Stdout)
					ui.PrintOnError("Rendering list", err)
				}
			}
		},
	}
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&noExecution, "no-execution", false, "don't show latest execution")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test crd")

	return cmd
}
