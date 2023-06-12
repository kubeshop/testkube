package testsuites

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites/renderer"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetTestSuiteCmd() *cobra.Command {
	var (
		selectors   []string
		noExecution bool
		crdOnly     bool
	)

	cmd := &cobra.Command{
		Use:     "testsuite <testSuiteName>",
		Aliases: []string{"testsuites", "ts"},
		Short:   "Get test suite by name",
		Long:    `Getting test suite from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			firstEntry := true
			if len(args) > 0 {
				ui.NL()
				name := args[0]
				testSuite, err := client.GetTestSuiteWithExecution(name)
				ui.ExitOnError("getting test suite "+name, err)

				if testSuite.TestSuite != nil {
					if crdOnly {
						testSuite.TestSuite.QuoteTestSuiteTextFields()
						common.UIPrintCRD(crd.TemplateTestSuite, testSuite.TestSuite, &firstEntry)
						return
					}

					ui.Info("Test Suite:")
					err = render.Obj(cmd, *testSuite.TestSuite, os.Stdout, renderer.TestSuiteRenderer)
					ui.ExitOnError("rendering obj", err)
				}

				if testSuite.LatestExecution != nil && !noExecution {
					ui.NL()
					ui.Info("Latest execution")
					err = render.Obj(cmd, *testSuite.LatestExecution, os.Stdout, renderer.TestSuiteExecutionRenderer)
					ui.ExitOnError("rendering obj", err)
				}
			} else {
				if noExecution {
					testSuites, err := client.ListTestSuites(strings.Join(selectors, ","))
					ui.ExitOnError("getting test suites", err)

					if crdOnly {
						for _, testSuite := range testSuites {
							testSuite.QuoteTestSuiteTextFields()
							common.UIPrintCRD(crd.TemplateTestSuite, testSuite, &firstEntry)
						}

						return
					}

					err = render.List(cmd, testSuites, os.Stdout)
					ui.ExitOnError("rendering list", err)
				} else {
					testSuites, err := client.ListTestSuiteWithExecutionSummaries(strings.Join(selectors, ","))
					ui.ExitOnError("getting test suite with execution summaries", err)

					if crdOnly {
						for _, testSuite := range testSuites {
							if testSuite.TestSuite != nil {
								testSuite.TestSuite.QuoteTestSuiteTextFields()
								common.UIPrintCRD(crd.TemplateTestSuite, testSuite.TestSuite, &firstEntry)
							}
						}

						return
					}

					err = render.List(cmd, testSuites, os.Stdout)
					ui.ExitOnError("rendering list", err)
				}
			}

		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&noExecution, "no-execution", false, "don't show latest execution")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test crd")

	return cmd
}
