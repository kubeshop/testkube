package tests

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteTestsCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete tests",
		Long:  `Delete tests`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			if len(args) == 0 {
				ui.ExitOnError("delete test", fmt.Errorf("test name is not specified"))
			}
			client, namespace := GetClient(cmd)

			name = args[0]
			err := client.DeleteTest(name, namespace)
			ui.ExitOnError("delete test "+name+" from namespace "+namespace, err)
			ui.Success("Succesfully deleted", name)
		},
	}

	return cmd
}
