package scripts

import (
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteScriptsCmd() *cobra.Command {
	var name string
	var deleteAll bool
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete scripts",
		Long:  `Delete scripts `,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			client, namespace := GetClient(cmd)
			var err error
			message := "delete all scripts from namespace " + namespace
			if deleteAll {
				err = client.DeleteScripts(namespace)
			} else {
				err = client.DeleteScript(name, namespace)
				message = "delete script " + name + " from namespace " + namespace
			}
			ui.ExitOnError(message, err)

			ui.Success("Succesfully deleted", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique script name")
	cmd.Flags().BoolVarP(&deleteAll, "all", "a", false, "delete all scripts")

	return cmd
}
