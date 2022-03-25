package testsuites

import (
	"os"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites/renderer"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetTestSuiteCmd() *cobra.Command {
	var selectors []string

	cmd := &cobra.Command{
		Use:     "testsuite <testSuiteName>",
		Aliases: []string{"testsuites", "ts"},
		Short:   "Get test suite by name",
		Long:    `Getting test suite from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			client, _ := common.GetClient(cmd)

			if len(args) > 0 {
				name := args[0]
				testSuite, err := client.GetTestSuite(name, namespace)
				ui.ExitOnError("getting test suite "+name, err)
				err = render.Obj(cmd, testSuite, os.Stdout, renderer.TestSuiteRenderer)
				ui.ExitOnError("rendering obj", err)
			} else {
				testSuites, err := client.ListTestSuites(namespace, strings.Join(selectors, ","))
				ui.ExitOnError("getting test suites", err)
				err = render.List(cmd, testSuites, os.Stdout)
				ui.ExitOnError("rendering list", err)
			}

		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	return cmd
}
