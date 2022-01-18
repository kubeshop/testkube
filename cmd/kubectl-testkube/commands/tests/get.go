package tests

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewGetTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <testName>",
		Aliases: []string{"g"},
		Short:   "Get test by name",
		Long:    `Getting test from given namespace - if no namespace given "testkube" namespace is used`,
		Args:    validator.TestName,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			ui.Logo()

			name := args[0]
			client, _ := common.GetClient(cmd)
			test, err := client.GetTest(name, namespace)
			ui.ExitOnError("getting test "+name, err)

			out, err := yaml.Marshal(test)
			ui.ExitOnError("getting yaml ", err)

			fmt.Printf("%s\n", out)
		},
	}
}
