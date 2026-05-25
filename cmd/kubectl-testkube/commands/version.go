package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Shows version and build info",
		Long:    `Shows version and build info`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			info, err := client.GetServerInfo()
			// Track the actual server version separately from the displayed
			// string so we can compare semantic versions even when the API
			// call failed. info.Version is reused as the display field and may
			// include an error suffix below.
			serverVersion := info.Version
			if err != nil {
				info.Version = info.Version + " " + err.Error()
				serverVersion = ""
			}

			ui.Logo()
			ui.Info("Client Version", common.Version)
			if info.Version != "" {
				ui.Info("Server Version", info.Version)
			}
			if info.Commit != "" {
				ui.Info("Commit", common.Commit)
			}
			if common.BuiltBy != "" {
				ui.Info("Built by", common.BuiltBy)
			}
			if common.Date != "" {
				ui.Info("Build date", common.Date)
			}

			cfg, err := config.Load()
			if err != nil {
				ui.WarnOnError("loading config for update check", err)
				return
			}
			if common.CheckComponentsStatus(cmd, &cfg, common.Version, serverVersion) {
				if err := config.Save(cfg); err != nil {
					ui.WarnOnError("saving update-check cache", err)
				}
			}
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)
			common.UiContextHeader(cmd, cfg)

			validator.PersistentPreRunVersionCheck(cmd, common.Version)
		},
	}
}
