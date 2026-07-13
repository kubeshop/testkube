package agents

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRotateRegistrationTokenCommand() *cobra.Command {
	var gracePeriod string
	var yes bool

	cmd := &cobra.Command{
		Use:   "rotate-registration-token [environment-id]",
		Short: "Rotate the registration token for an environment",
		Args:  cobra.MaximumNArgs(1),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)
			common.UiContextHeader(cmd, cfg)
			validator.PersistentPreRunVersionCheck(cmd, common.Version)
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)

			envID := cfg.CloudContext.EnvironmentId
			if len(args) == 1 {
				envID = args[0]
			}

			if !yes && !ui.Confirm(fmt.Sprintf("Rotate registration token for environment ID '%s'?", envID)) {
				return
			}

			client := cloudclient.NewEnvironmentsClient(cfg.CloudContext.ApiUri, cfg.CloudContext.ApiKey, cfg.CloudContext.OrganizationId, cfg.SkipTLS || cfg.CloudContext.SkipTLS)
			result, err := client.RotateRegistrationToken(cmd.Context(), envID, gracePeriod)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrAgentRotateRegistrationTokenFailed,
					"Failed to rotate environment registration token",
					"Verify the environment ID is correct, your credentials are valid, and your user is an organization admin or owner.",
					err,
				))
				return
			}

			ui.Success("Registration token rotated successfully")
			fmt.Println()
			ui.Warn("New Registration Token:", result.RegistrationToken)
			ui.Warn("Grace Period:          ", result.GracePeriod)
			ui.Warn("Old Token Expires:      ", result.OldTokenExpiresAt.In(time.Local).Format(time.RFC822Z))
			fmt.Println()
			ui.Info("Update runner.register.token or the Secret referenced by runner.register.tokenSecret before the old token expires.")
		},
	}

	cmd.Flags().StringVar(&gracePeriod, "grace-period", "24h", "how long the old token remains valid for registration (maximum 168h; use 0s for immediate rotation)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
