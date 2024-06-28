package agent

import (
	"github.com/spf13/cobra"
)

func NewDebugAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "debug",
		Short:      "Debug Agent info",
		Deprecated: "use `testkube debug agent` instead",
	}

	return cmd
}
