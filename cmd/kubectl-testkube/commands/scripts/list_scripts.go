package scripts

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

var tags []string

func NewListScriptsCmd() *cobra.Command {
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
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "--tags 1,2,3")

	return cmd
}
