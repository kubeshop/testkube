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
	cmd := controlplaneresources.NewApplyCmd(controlplaneresources.NewClient)
	cmd.Annotations = map[string]string{cmdGroupAnnotation: cmdGroupCommands}
	versionCheck := unlessDryRun(func(cmd *cobra.Command, _ []string) {
		validator.PersistentPreRunVersionCheck(cmd, common.Version)
	})
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		ui.ExitOnError("loading config", err)
		common.UiContextHeader(cmd, cfg)
		versionCheck(cmd, args)
	}
	cmd.PersistentPostRun = unlessDryRun(func(cmd *cobra.Command, args []string) {
		if RootCmd.PersistentPostRun != nil {
			RootCmd.PersistentPostRun(cmd, args)
		}
	})
	return cmd
}

// unlessDryRun skips a hook that connects to the Control Plane, which can
// refresh the token or start a login, when the command is a dry run: that
// validates offline. The version check and the root post-run hook both
// connect.
func unlessDryRun(hook func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
			return
		}
		hook(cmd, args)
	}
}
