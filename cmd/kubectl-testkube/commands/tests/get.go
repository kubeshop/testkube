package tests

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewGetTestsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get test by name",
		Long:  `Getting test from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()
			ui.Logo()

			if len(args) == 0 {
				ui.Failf("test name is not specified")
			}

			name := args[0]
			client, _ := GetClient(cmd)
			test, err := client.GetTest(name, namespace)
			ui.ExitOnError("getting test "+name, err)

			out, err := yaml.Marshal(test)
			ui.ExitOnError("getting yaml ", err)

			fmt.Printf("%s\n", out)
		},
	}
}
