package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/executors"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <resourceName>",
		Aliases: []string{"u"},
		Short:   "Update resource",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// version validation
			// if client version is less than server version show warning
			client, _ := common.GetClient(cmd)

			err := ValidateVersions(client)
			if err != nil {
				ui.Warn(err.Error())
			}
		},
	}

	cmd.AddCommand(tests.NewUpdateTestsCmd())
	cmd.AddCommand(testsuites.NewUpdateTestSuitesCmd())
	cmd.AddCommand(executors.NewCreateExecutorCmd())
	cmd.AddCommand(tests.NewCreateTestsCmd())
	cmd.AddCommand(testsuites.NewCreateTestSuitesCmd())

	return cmd
}
