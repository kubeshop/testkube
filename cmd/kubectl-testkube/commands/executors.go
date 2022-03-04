package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/executors"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewExecutorsCmd() *cobra.Command {
	var (
		client    string
		verbose   bool
		namespace string
	)

	cmd := &cobra.Command{
		Use:   "executors",
		Short: "Executor management commands",
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

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "testkube", "kubernetes namespace")

	cmd.AddCommand(executors.NewCreateExecutorCmd())
	cmd.AddCommand(executors.NewGetExecutorCmd())
	cmd.AddCommand(executors.NewListExecutorCmd())
	cmd.AddCommand(executors.NewDeleteExecutorCmd())

	return cmd
}
