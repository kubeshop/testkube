package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/executors"
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
		Long:  `All available scripts and scripts executions commands`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "testkube", "kubernetes namespace")

	// output renderer flags
	cmd.PersistentFlags().StringP("output", "o", "raw", "output type one of raw|json|go ")
	cmd.PersistentFlags().StringP("go-template", "", "{{ . | printf \"%+v\"  }}", "in case of choosing output==go pass golang template")

	cmd.AddCommand(executors.NewCreateExecutorCmd())
	cmd.AddCommand(executors.NewGetExecutorCmd())
	cmd.AddCommand(executors.NewListExecutorCmd())
	cmd.AddCommand(executors.NewDeleteExecutorCmd())

	return cmd
}
