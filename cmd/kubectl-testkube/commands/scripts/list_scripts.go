package scripts

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListScriptsCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get all available scripts",
		Long:  `Getting all available scritps from given namespace - if no namespace given "testkube" namespace is used`,
		Run: func(cmd *cobra.Command, args []string) {
			namespace := cmd.Flag("namespace").Value.String()

			client, _ := GetClient(cmd)
			scripts, err := client.ListScripts(namespace, tags)
			ui.ExitOnError("getting all scripts in namespace "+namespace, err)

			ui.Table(scripts, os.Stdout)
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "--tags 1,2,3")

	return cmd
}
