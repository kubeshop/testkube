package agent

import (
	"github.com/spf13/cobra"
)

func NewDebugAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "debug",
		Short:      "Debug Runner info",
		Deprecated: "use `testkube debug runner` instead",
	}

	return cmd
}
