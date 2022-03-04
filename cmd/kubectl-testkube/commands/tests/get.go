package tests

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewGetTestsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "tests <testName>",
		Aliases: []string{"test", "t"},
		Short:   "Get test by name",
		Long:    `Getting test from given namespace - if no namespace given "testkube" namespace is used`,
		Args:    validator.TestName,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			name := args[0]
			namespace := cmd.Flag("namespace").Value.String()

			client, _ := common.GetClient(cmd)

			test, err := client.GetTest(name, namespace)
			ui.ExitOnError("getting test "+name, err)

			out, err := yaml.Marshal(test)
			ui.ExitOnError("getting yaml ", err)

			fmt.Printf("%s\n", out)
		},
	}
}
