package testsuites

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewGetTestSuiteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "testsuite <testSuiteName>",
		Aliases: []string{"testsuites"},
		Short:   "Get test suite by name",
		Long:    `Getting test suite from given namespace - if no namespace given "testkube" namespace is used`,
		Args:    validator.TestSuiteName,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			ui.Logo()

			name := args[0]
			client, _ := common.GetClient(cmd)
			testSuite, err := client.GetTestSuite(name, namespace)
			ui.ExitOnError("getting test "+name, err)

			out, err := yaml.Marshal(testSuite)
			ui.ExitOnError("getting yaml ", err)

			fmt.Printf("%s\n", out)
		},
	}
}
