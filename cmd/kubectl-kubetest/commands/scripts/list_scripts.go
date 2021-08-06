package scripts

import (
	"os"

	"github.com/kubeshop/kubetest/pkg/api/client"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {
	ListScriptsCmd.Flags().String("namespace", "default", "script type (defaults to postman-collection)")
}

var ListScriptsCmd = &cobra.Command{
	Use:   "list",
	Short: "Get all available scripts",
	Long:  `Getting all available scritps from given namespace - if no namespace given "default" namespace is used`,
	Run: func(cmd *cobra.Command, args []string) {

		namespace := cmd.Flag("namespace").Value.String()
		client := client.NewDefaultScriptsAPI()

		scripts, err := client.ListScripts(namespace)
		ui.ExitOnError("getting all scripts in namespace "+namespace, err)

		ui.Table(scripts, os.Stdout)
	},
}
