package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/executors"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/tests"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsuites"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/webhooks"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <resourceName>",
		Aliases: []string{"g"},
		Short:   "Get resources",
		Long:    `Get available resources, get single item or list`,
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

	// output renderer flags
	cmd.PersistentFlags().StringP("output", "o", "raw", "output type one of raw|json|go ")
	cmd.PersistentFlags().StringP("go-template", "", "{{ . | printf \"%+v\"  }}", "in case of choosing output==go pass golang template")

	cmd.AddCommand(tests.NewGetTestsCmd())
	cmd.AddCommand(testsuites.NewGetTestSuiteCmd())
	cmd.AddCommand(webhooks.NewGetWebhookCmd())
	cmd.AddCommand(executors.NewGetExecutorCmd())
	cmd.AddCommand(tests.NewGetExecutionCmd())

	return cmd
}
