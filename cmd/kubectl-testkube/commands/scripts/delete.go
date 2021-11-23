package scripts

import (
	"fmt"

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
			if len(args) == 0 && !deleteAll {
				ui.ExitOnError("delete script", fmt.Errorf("script name is not specified"))
			}
			client, namespace := GetClient(cmd)
			var err error
			message := "delete all scripts from namespace " + namespace
			if deleteAll {
				err = client.DeleteScripts(namespace)
			} else {
				name = args[0]
				err = client.DeleteScript(name, namespace)
				message = "delete script " + name + " from namespace " + namespace
			}
			ui.ExitOnError(message, err)

			ui.Success("Succesfully deleted", name)
		},
	}

	cmd.Flags().BoolVarP(&deleteAll, "all", "a", false, "delete all scripts")

	return cmd
}
