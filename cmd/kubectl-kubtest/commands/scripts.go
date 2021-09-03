package commands

import (
	"github.com/kubeshop/kubtest/cmd/kubectl-kubtest/commands/scripts"
	"github.com/kubeshop/kubtest/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	client    string
	verbose   bool
	namespace string
)

func NewScriptsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scripts",
		Short: "Scripts management commands",
		Long:  `All available scripts and scripts executions commands`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ui.VerboseMode = verbose
		},
	}

	cmd.PersistentFlags().StringVarP(&client, "client", "c", "proxy", "Client used for connecting to kubtest API one of proxy|direct")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "should I show additional debug messages")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "s", "default", "kubernetes namespace")

	// output renderer flags
	cmd.PersistentFlags().StringP("output", "o", "raw", "output typoe one of raw|json|go ")
	cmd.PersistentFlags().StringP("go-template", "", "{{ . | printf \"%+v\"  }}", "in case of choosing output==go pass golang template")

	cmd.AddCommand(scripts.NewAbortExecutionCmd())
	cmd.AddCommand(scripts.NewListScriptsCmd())
	cmd.AddCommand(scripts.NewStartScriptCmd())
	cmd.AddCommand(scripts.NewGetScriptExecutionCmd())
	cmd.AddCommand(scripts.NewWatchScriptExecutionCmd())
	cmd.AddCommand(scripts.NewListScriptExecutionsCmd())
	cmd.AddCommand(scripts.NewCreateScriptsCmd())
	return cmd
}
