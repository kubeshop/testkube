package tests

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListTestsCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "Get all available tests",
		Long:    `Getting all available tests from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()

			client, _ := GetClient(cmd)
			tests, err := client.ListTests(namespace, tags)

			ui.ExitOnError("getting all tests in namespace "+namespace, err)

			ui.Table(tests, os.Stdout)
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "comma separated list of tags: --tags tag1,tag2,tag3")
	return cmd
}
