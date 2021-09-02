package commands

import (
	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Shows version and build info",
		Long:  `Shows version and build info`,
		Run: func(cmd *cobra.Command, args []string) {

			ui.Logo()
			ui.Info("Version", Version)
			ui.Info("Commit", Commit)
			ui.Info("Built by", BuiltBy)
			ui.Info("Build date", Date)

		},
	}
}
