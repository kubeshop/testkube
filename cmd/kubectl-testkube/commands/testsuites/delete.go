package testsuites

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteTestSuiteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <testSuiteName>",
		Short: "Delete test suite",
		Long:  `Delete test suite by name`,
		Args:  validator.TestSuiteName,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			client, namespace := common.GetClient(cmd)

			name := args[0]
			err := client.DeleteTestSuite(name, namespace)
			ui.ExitOnError("delete test "+name+" from namespace "+namespace, err)
			ui.Success("Succesfully deleted", name)
		},
	}

	return cmd
}
