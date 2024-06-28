package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/debug"
)

// NewDebugCmd creates the 'testkube debug' command
func NewDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "debug",
		Aliases: []string{"dbg", "d"},
		Short:   "Print debugging info",
	}

	cmd.AddCommand(debug.NewDebugOssCmd())
	cmd.AddCommand(debug.NewDebugAgentCmd())
	cmd.AddCommand(debug.NewDebugControlPlaneCmd())

	return cmd
}
