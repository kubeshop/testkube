package tests

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListExecutionsCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "executions",
		Short: "Get all test executions",
		Long:  `Getting all test executions`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()

			client, _ := GetClient(cmd)
			executions, err := client.ListTestExecutions(namespace, 1000, tags)

			ui.ExitOnError("getting all tests executions in namespace "+namespace, err)

			ui.Table(executions, os.Stdout)
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "comma separated list of tags: --tags tag1,tag2,tag3")
	return cmd
}
