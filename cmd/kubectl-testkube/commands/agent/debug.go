package agent

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
)

func NewDebugAgentCmd() *cobra.Command {
	var show common.CommaList

	cmd := &cobra.Command{
		Use:        "debug",
		Short:      "Debug Agent info",
		Deprecated: "use `testkube debug agent` instead",
	}

	cmd.Flags().Var(&show, "show", "Comma-separated list of features to show, one of: pods,services,storageclasses,api,worker,ui,dex,nats,mongo,minio, defaults to all")

	return cmd
}
