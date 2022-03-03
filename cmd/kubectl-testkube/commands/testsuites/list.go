package testsuites

import (
	"os"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListTestSuitesCmd() *cobra.Command {
	var selectors []string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "Get all available test suites",
		Long:    `Getting all available test suites from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()

			client, _ := common.GetClient(cmd)
			tests, err := client.ListTestSuites(namespace, strings.Join(selectors, ","))

			ui.ExitOnError("getting all test suites in namespace "+namespace, err)

			ui.Table(tests, os.Stdout)
		},
	}
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	return cmd
}
