package scripts

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteAllScriptsCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete-all",
		Short: "Delete all scripts",
		Args:  validator.ScriptName,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			namespace := cmd.Flag("namespace").Value.String()

			client, _ := common.GetClient(cmd)
			err := client.DeleteScripts(namespace)
			ui.ExitOnError("delete all scripts from namespace "+namespace, err)

			ui.Success("Succesfully deleted", name)
		},
	}

	return cmd
}
