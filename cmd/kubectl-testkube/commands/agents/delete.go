package agents

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	common2 "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteAgentCommand() *cobra.Command {
	var (
		uninstall, noUninstall     bool
		deleteAgent, noDeleteAgent bool
	)
	cmd := &cobra.Command{
		Use:     "runner",
		Aliases: []string{"agent"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !uninstall && !noUninstall {
				uninstall = ui.Confirm("should it uninstall runner?")
			}
			if !deleteAgent && !noDeleteAgent {
				deleteAgent = ui.Confirm("should it delete runner in the Control Plane?")
			}
			UiDeleteAgent(cmd, args[0], uninstall, deleteAgent)
		},
	}

	cmd.Flags().BoolVarP(&uninstall, "uninstall", "u", false, "should it uninstall the runner too")
	cmd.Flags().BoolVarP(&noUninstall, "no-uninstall", "U", false, "should it keep the runner installed")
	cmd.Flags().BoolVarP(&deleteAgent, "delete", "d", false, "should it delete runner in the Control Plane")
	cmd.Flags().BoolVarP(&noDeleteAgent, "no-delete", "D", false, "should it keep the runner definition in the Control Plane")

	return cmd
}

func NewDeleteCRDCommand() *cobra.Command {
	var (
		confirm bool
	)
	cmd := &cobra.Command{
		Use:  "crd",
		Args: cobra.MaximumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if !confirm && !ui.Confirm("should it uninstall CRDs?") {
				os.Exit(1)
			}

			UiUninstallCRD(cmd)
		},
	}

	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "non-interactive confirmation")

	return cmd
}

func UiUninstallCRD(cmd *cobra.Command) {
	spinner := ui.NewSpinner("Fetching current CRDs")
	currentNamespace, currentReleaseName, installed, err := GetCRDInstallation()
	if err != nil {
		spinner.Fail()
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrResourceLookupFailed,
			"Error getting the installed CRDs",
			common2.ClusterLookupHint,
			err,
		))
	}

	if installed && currentReleaseName == "" {
		spinner.Fail()
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrInvalidInstallConfig,
			"The CRDs are not managed by the Testkube Helm Chart",
			"Delete the Testkube CRDs by hand, or install them with `testkube install crd` so that Helm owns them",
			errors.New("the CRDs are installed, but they carry no Helm release annotation"),
		))
	}

	if installed {
		spinner.Success(fmt.Sprintf("The CRDs are installed in '%s' namespace", currentNamespace))
	} else {
		spinner.Success("CRDs not found")
		os.Exit(0)
	}

	spinner = ui.NewSpinner("Uninstalling CRDs")
	common2.HandleCLIError(common2.HelmUninstall(currentNamespace, currentReleaseName))
	spinner.Success()
}

func UiDeleteAgent(cmd *cobra.Command, name string, uninstall, deleteAgent bool) {
	agent, err := GetControlPlaneAgent(cmd, name)
	if err != nil {
		common2.HandleCLIError(common2.NewCLIError(
			common2.TKErrAgentGetFailed,
			"Error getting the runner",
			common2.AgentLookupHint,
			err,
		))
	}

	// Uninstall the Runner
	if uninstall {
		var nses []string
		if agent.Namespace != "" {
			nses = append(nses, agent.Namespace)
		} else {
			nses, err = GetKubernetesNamespaces()
			if err != nil {
				common2.HandleCLIError(common2.NewCLIError(
					common2.TKErrResourceLookupFailed,
					"Error listing the Kubernetes namespaces",
					common2.ClusterLookupHint,
					err,
				))
			}
		}

		agents, err := GetKubernetesAgents(nses)
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrResourceLookupFailed,
				"Error getting the runners running in the cluster",
				common2.ClusterLookupHint,
				err,
			))
		}

		var kubernetesAgent *internalAgent
		for i := range agents {
			if agents[i].AgentID.Value == agent.ID {
				kubernetesAgent = &agents[i]
				break
			}
		}
		if kubernetesAgent == nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrResourceNotFound,
				"Runner not installed in the cluster",
				"Pass '--no-uninstall' to delete the runner in the Control Plane only, or check that your kubeconfig points at the cluster the runner runs in",
				fmt.Errorf("kubernetes runner not found: namespaces: %s", strings.Join(nses, ", ")),
			))
			return
		}

		spinner := ui.NewSpinner("Running Helm command...")
		common2.HandleCLIError(common2.HelmUninstall(kubernetesAgent.Pod.Namespace, fmt.Sprintf("testkube-%s", agent.Name)))
		spinner.Success()
	}

	// Delete the Agent
	if deleteAgent {
		spinner := ui.NewSpinner("Deleting runner in the Control Plane...")
		err := DeleteControlPlaneAgent(cmd, agent.ID)
		if err != nil {
			common2.HandleCLIError(common2.NewCLIError(
				common2.TKErrAgentWriteFailed,
				"Error deleting the runner",
				common2.AgentWriteHint,
				err,
			))
		}
		spinner.Success()
	}
}
