package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/context"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "set <resourceName>",
		Aliases:     []string{"s"},
		Short:       "Set resources",
		Long:        `Set available resources, like context etc`,
		Annotations: map[string]string{cmdGroupAnnotation: cmdGroupCommands},
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			validator.PersistentPreRunVersionCheck(cmd, common.Version)
		}}

	cmd.AddCommand(context.NewSetContextCmd())

	return cmd
}
