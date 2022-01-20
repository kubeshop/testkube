package tests

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteTestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-all",
		Short: "Delete all tests in namespace",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			client, namespace := common.GetClient(cmd)

			err := client.DeleteTests(namespace)
			ui.ExitOnError("delete all tests from namespace "+namespace, err)
			ui.Success("Succesfully deleted all tests in namespace", namespace)
		},
	}

	return cmd
}
