package commands

import (
	"github.com/kubeshop/kubetest/cmd/kubectl-kubetest/commands/scripts"
	"github.com/spf13/cobra"
)

func init() {
	scriptsCmd.AddCommand(scripts.AbortExecutionCmd)
	scriptsCmd.AddCommand(scripts.ListScriptsCmd)
	scriptsCmd.AddCommand(scripts.StartScriptCmd)
	scriptsCmd.AddCommand(scripts.GetScriptExecutionCmd)
	scriptsCmd.AddCommand(scripts.GetScriptExecutionsCmd)
	RootCmd.AddCommand(scriptsCmd)
}

var scriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: "Scripts management commands",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
