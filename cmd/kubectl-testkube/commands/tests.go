package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/spf13/cobra"
)

func NewTestsCmd() *cobra.Command {
	var (
		client    string
		verbose   bool
		namespace string
	)
	cmd := &cobra.Command{
		Use:     "tests",
		Aliases: []string{"test", "t"},
		Short:   "Tests management commands",
		Long:    `All available tests and tests executions commands`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// version validation
			// if client version is less than server version show warning
			client, _ := tests.GetClient(cmd)

			err := ValidateVersions(client)
			if err != nil {
				ui.Warn(err.Error())
			}
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to testkube API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "testkube", "kubernetes namespace")

	// output renderer flags
	cmd.PersistentFlags().StringP("output", "o", "raw", "output type one of raw|json|go ")
	cmd.PersistentFlags().StringP("go-template", "", "{{ . | printf \"%+v\"  }}", "in case of choosing output==go pass golang template")

	cmd.AddCommand(tests.NewListTestsCmd())
	cmd.AddCommand(tests.NewGetTestCmd())
	cmd.AddCommand(tests.NewStartTestCmd())
	cmd.AddCommand(tests.NewCreateTestsCmd())
	cmd.AddCommand(tests.NewUpdateTestsCmd())
	cmd.AddCommand(tests.NewDeleteTestsCmd())
	cmd.AddCommand(tests.NewTestExecutionCmd())
	cmd.AddCommand(tests.NewWatchTestExecutionCmd())
	cmd.AddCommand(tests.NewTestExecutionsCmd())
	return cmd
}
