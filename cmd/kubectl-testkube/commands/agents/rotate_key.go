package agents

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRotateKeyCommand() *cobra.Command {
	var (
		gracePeriod string
		yes         bool
	)

	cmd := &cobra.Command{
		Use:   "rotate-key <nameOrId>",
		Short: "Rotate the secret key for an agent",
		Long:  "Rotate the secret key for an agent with a configurable grace period during which the old key remains valid",
		Args:  cobra.ExactArgs(1),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrConfigInitFailed,
					"Error loading testkube config file",
					common.ConfigFileHint,
					err,
				))
			}
			common.UiContextHeader(cmd, cfg)
			validator.PersistentPreRunVersionCheck(cmd, common.Version)
		},
		Run: func(cmd *cobra.Command, args []string) {
			nameOrID := args[0]

			// Validate the agent exists
			agent, err := GetControlPlaneAgent(cmd, nameOrID)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrAgentGetFailed,
					"Error getting the agent",
					"Check the agent name or ID and that your credentials are valid, or list the agents with `testkube get agents`",
					err,
				))
				return
			}

			// Confirm unless --yes
			if !yes {
				ok := ui.Confirm(fmt.Sprintf("Rotate secret key for agent '%s'?", agent.Name))
				if !ok {
					return
				}
			}

			// Rotate the key
			result, err := RotateControlPlaneAgentKey(cmd, agent.ID, gracePeriod)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrAgentRotateKeyFailed,
					"Error rotating the agent secret key",
					"Check that your credentials are valid and that the '--grace-period' value is one the control plane accepts, for example 24h or 0s",
					err,
				))
				return
			}

			// Display results
			ui.Success("Secret key rotated successfully")
			fmt.Println()
			ui.Warn("Agent:         ", agent.Name)
			ui.Warn("New Secret Key:", result.SecretKey)
			if result.GracePeriod != "" {
				ui.Warn("Grace Period:  ", result.GracePeriod)
			} else {
				ui.Warn("Grace Period:  ", gracePeriod)
			}
			if result.OldKeyExpiresAt != nil {
				ui.Warn("Old Key Expires:", result.OldKeyExpiresAt.In(time.Local).Format(time.RFC822Z))
			}

			fmt.Println()
			ui.Info("To update the agent's Kubernetes secret, run:")
			fmt.Printf("  kubectl create secret generic testkube-agent-secret --from-literal=TESTKUBE_PRO_API_KEY=%s --dry-run=client -o yaml | kubectl apply -f -\n", result.SecretKey)
			fmt.Println()
		},
	}

	cmd.Flags().StringVar(&gracePeriod, "grace-period", "24h", "how long the old key remains valid (e.g., 24h, 48h, 0s for immediate)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")

	return cmd
}
