package agents

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	common2 "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteAgentCommand() *cobra.Command {
	var (
		uninstall, noUninstall     bool
		deleteAgent, noDeleteAgent bool
	)
	cmd := &cobra.Command{
		Use:     "agent",
		Aliases: []string{"runner", "gitops"},
		Args:    cobra.ExactArgs(1),
		Hidden:  !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			if !uninstall && !noUninstall {
				should := ui.Select("should it uninstall agent?", []string{"yes", "no"})
				if should == "" {
					ui.Failf("you need to decide whether it should uninstall or not")
				}
				uninstall = should == "yes"
			}
			if !deleteAgent && !noDeleteAgent {
				should := ui.Select("should it delete agent in the Control Plane?", []string{"yes", "no"})
				if should == "" {
					ui.Failf("you need to decide whether it should delete agent or not")
				}
				deleteAgent = should == "yes"
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
