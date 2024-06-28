package debug

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	defaultOssNamespace = "testkube"
)

func NewDebugOssCmd() *cobra.Command {
	var show common.CommaList

	cmd := &cobra.Command{
		Use:   "oss",
		Short: "Show OSS installation debug info",
		Run:   RunDebugOssCmdFunc(&show),
	}

	return cmd
}

func RunDebugOssCmdFunc(show *common.CommaList) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		client, _, err := common.GetClient(cmd)
		ui.ExitOnError("getting client", err)

		d, err := GetDebugInfo(client)
		ui.ExitOnError("get debug info", err)

		PrintDebugInfo(d)
	}
}
