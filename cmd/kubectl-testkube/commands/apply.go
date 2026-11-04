package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/controlplaneresources"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewApplyCmd returns `testkube apply`, which creates or updates Control
// Plane resources from YAML.
func NewApplyCmd() *cobra.Command {
	cmd := controlplaneresources.NewApplyCmd()
	cmd.Annotations = map[string]string{cmdGroupAnnotation: cmdGroupCommands}
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		ui.ExitOnError("loading config", err)
		common.UiContextHeader(cmd, cfg)

		validator.PersistentPreRunVersionCheck(cmd, common.Version)
	}
	return cmd
}
