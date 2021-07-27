package scripts

import (
	"os"

	"github.com/kubeshop/kubetest/pkg/api/client"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

var ListScriptsCmd = &cobra.Command{
	Use:   "list",
	Short: "Get all available scripts",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ns := "default"
		if len(args) == 1 {
			ns = args[0]
		}

		client := client.NewRESTClient(client.DefaultURI)
		scripts, err := client.ListScripts(ns)
		ui.ExitOnError("getting all scripts in ns="+ns, err)
		ui.Table(scripts, os.Stdout)

	},
}
