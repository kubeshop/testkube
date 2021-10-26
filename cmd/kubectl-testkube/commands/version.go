package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/scripts"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Shows version and build info",
		Long:  `Shows version and build info`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := scripts.GetClient(cmd)
			info, err := client.GetServerInfo()
			if err != nil {
				info.Version = info.Version + " " + err.Error()
			}

			ui.Logo()
			ui.Info("Client Version", Version)
			ui.Info("Server Version", info.Version)
			ui.Info("Commit", Commit)
			ui.Info("Built by", BuiltBy)
			ui.Info("Build date", Date)

		},
	}
}
