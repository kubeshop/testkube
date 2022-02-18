package testsuites

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteTestSuitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-all",
		Short: "Delete all test suites in namespace",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			client, namespace := common.GetClient(cmd)

			err := client.DeleteTestSuites(namespace)
			ui.ExitOnError("delete all tests from namespace "+namespace, err)
			ui.Success("Succesfully deleted all test suites in namespace", namespace)
		},
	}

	return cmd
}
