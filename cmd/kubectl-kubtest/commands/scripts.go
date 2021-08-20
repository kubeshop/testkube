package commands

import (
	"github.com/kubeshop/kubtest/cmd/kubectl-kubtest/commands/scripts"
	"github.com/kubeshop/kubtest/pkg/ui"

	"github.com/spf13/cobra"
)

func init() {
	scriptsCmd.PersistentFlags().StringP("client", "c", "proxy", "Client used for connecting to kubtest API one of proxy|direct")
	scriptsCmd.PersistentFlags().BoolP("verbose", "v", false, "should I show additional debug messages")
	scriptsCmd.PersistentFlags().StringP("namespace", "s", "default", "kubernetes namespace")

	// output renderer flags
	scriptsCmd.PersistentFlags().StringP("output", "o", "raw", "output typoe one of raw|json|go ")
	scriptsCmd.PersistentFlags().StringP("go-template", "", "{{ . | printf \"%+v\"  }}", "in case of choosing output==go pass golang template")

	scriptsCmd.AddCommand(scripts.AbortExecutionCmd)
	scriptsCmd.AddCommand(scripts.ListScriptsCmd)
	scriptsCmd.AddCommand(scripts.StartScriptCmd)
	scriptsCmd.AddCommand(scripts.GetScriptExecutionCmd)
	scriptsCmd.AddCommand(scripts.ListScriptExecutionsCmd)
	scriptsCmd.AddCommand(scripts.CreateScriptsCmd)
	RootCmd.AddCommand(scriptsCmd)
}

var scriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: "Scripts management commands",
	Long:  `All available scripts and scripts executions commands`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose, err := cmd.Flags().GetBool("verbose"); !verbose && err == nil {
			ui.QuietMode = true
		}
	},
}
