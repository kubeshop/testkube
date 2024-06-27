package agent

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/debug"
)

func NewDebugAgentCmd() *cobra.Command {
	var show common.CommaList

	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug Agent info",
		Run:   debug.RunDebugAgentCmdFunc(&show),
	}

	cmd.Flags().Var(&show, "show", "Comma-separated list of features to show, one of: pods,services,storageclasses,api,worker,ui,dex,nats,mongo,minio, defaults to all")

	return cmd
}
