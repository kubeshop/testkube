package agents

import (
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
		Use:     "agent",
		Aliases: []string{"runner"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !uninstall && !noUninstall {
				uninstall = ui.Confirm("should it uninstall agent?")
			}
			if !deleteAgent && !noDeleteAgent {
				deleteAgent = ui.Confirm("should it delete agent in the Control Plane?")
			}
			UiDeleteAgent(cmd, args[0], uninstall, deleteAgent)
		},
	}

	cmd.Flags().BoolVarP(&uninstall, "uninstall", "u", false, "should it uninstall the agent too")
	cmd.Flags().BoolVarP(&noUninstall, "no-uninstall", "U", false, "should it keep the agent installed")
	cmd.Flags().BoolVarP(&deleteAgent, "delete", "d", false, "should it delete agent in the Control Plane")
	cmd.Flags().BoolVarP(&noDeleteAgent, "no-delete", "D", false, "should it keep the agent definition in the Control Plane")

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
		spinner.Fail(err)
		os.Exit(1)
	}

	if installed && currentReleaseName == "" {
		spinner.Fail("The CRDs are installed, but they are not managed by our Helm Chart")
		os.Exit(1)
	}

	if installed {
		spinner.Success(fmt.Sprintf("The CRDs are installed in '%s' namespace", currentNamespace))
	} else {
		spinner.Success("CRDs not found")
		os.Exit(0)
	}

	spinner = ui.NewSpinner("Uninstalling CRDs")
	cliErr := common2.HelmUninstall(currentNamespace, currentReleaseName)
	if cliErr != nil {
		cliErr.Print()
		os.Exit(1)
	}
	spinner.Success()
}

func UiDeleteAgent(cmd *cobra.Command, name string, uninstall, deleteAgent bool) {
	agent, err := GetControlPlaneAgent(cmd, name)
	ui.ExitOnError("getting agent", err)

	// Uninstall the Agent
	if uninstall {
		var nses []string
		if agent.Namespace != "" {
			nses = append(nses, agent.Namespace)
		} else {
			nses, err = GetKubernetesNamespaces()
			ui.ExitOnError("getting namespaces", err)
		}

		agents, err := GetKubernetesAgents(nses)
		ui.ExitOnError("getting agents", err)

		var kubernetesAgent *internalAgent
		for i := range agents {
			if agents[i].AgentID.Value == agent.ID {
				kubernetesAgent = &agents[i]
				break
			}
		}
		if kubernetesAgent == nil {
			ui.Failf("kubernetes agent not found: namespaces: %s", strings.Join(nses, ", "))
			return
		}

		spinner := ui.NewSpinner("Running Helm command...")
		cliErr := common2.HelmUninstall(kubernetesAgent.Pod.Namespace, fmt.Sprintf("testkube-%s", agent.Name))
		if cliErr != nil {
			cliErr.Print()
			os.Exit(1)
		}
		spinner.Success()
	}

	// Delete the Agent
	if deleteAgent {
		spinner := ui.NewSpinner("Deleting agent in the Control Plane...")
		err := DeleteControlPlaneAgent(cmd, agent.ID)
		ui.ExitOnError("deleting agent", err)
		spinner.Success()
	}
}
