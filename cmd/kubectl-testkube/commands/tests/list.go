package tests

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListTestsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Get all available tests",
		Long:  `Getting all available tests from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()

			client, _ := GetClient(cmd)

			tests, err := client.ListTests(namespace)
			ui.ExitOnError("getting all tests in namespace "+namespace, err)

			ui.Table(tests, os.Stdout)
		},
	}
}
