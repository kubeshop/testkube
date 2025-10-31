package agents

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetAgentCommand() *cobra.Command {
	var (
		decryptSecretKey bool
		showUnknown      bool
		showDeleted      bool
		allEnvironments  bool
	)
	cmd := &cobra.Command{
		Args:    cobra.MaximumNArgs(1),
		Use:     "agent [name]",
		Aliases: []string{"agents", "a"},
		PreRun: func(cmd *cobra.Command, args []string) {
			// Warn if incompatible flags are used together
			if allEnvironments && showUnknown {
				ui.Warn("Note: --all-environments is ignored when using --show-unknown (unknown agents have no environment registration)")
				allEnvironments = false
			}
			if allEnvironments && showDeleted {
				ui.Warn("Note: --all-environments is ignored when using --show-deleted (showing deleted agents only)")
				allEnvironments = false
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				UiListAgents(cmd, showUnknown, showDeleted, allEnvironments)
			} else {
				UiGetAgent(cmd, args[0], decryptSecretKey)
			}
		},
	}

	cmd.Flags().BoolVar(&decryptSecretKey, "decrypted-secret", false, "should it fetch decrypted secret key")
	cmd.Flags().BoolVar(&showUnknown, "show-unknown", false, "show only unknown agents (agents in cluster not registered in control plane)")
	cmd.Flags().BoolVar(&showDeleted, "show-deleted", false, "show only deleted agents")
	cmd.Flags().BoolVar(&allEnvironments, "all-environments", false, "show agents from all environments (not just current environment)")

	return cmd
}

func UiGetAgent(cmd *cobra.Command, agentId string, decryptSecretKey bool) {
	registeredAgents, err := GetControlPlaneAgents(cmd, true)
	ui.ExitOnError("getting agents", err)

	namespaces, err := GetKubernetesNamespaces()
	ui.ExitOnError("listing namespaces", err)

	agents, err := GetKubernetesAgents(namespaces)
	ui.ExitOnError("listing pods", err)

	agents = CombineAgents(agents, registeredAgents)

	var agent *internalAgent
	for _, a := range agents {
		if a.AgentID.Value == agentId || (a.Registered != nil && (a.Registered.ID == agentId || a.Registered.Name == agentId)) {
			agent = &a
			break
		}
	}
	if agent == nil {
		ui.Fail(fmt.Errorf("agent '%s' not found", agentId))
	}

	if decryptSecretKey {
		secretKey, err := GetControlPlaneAgentSecretKey(cmd, agent.Registered.ID)
		ui.ExitOnError("failed to decrypt secret key", err)
		agent.Registered.SecretKey = secretKey
	}

	PrintControlPlaneAgent(*agent.Registered)
}

func UiListAgents(cmd *cobra.Command, showUnknown bool, showDeleted bool, allEnvironments bool) {
	registeredAgents, err := GetControlPlaneAgents(cmd, showDeleted)
	ui.ExitOnError("getting agents", err)

	// Filter agents by current environment (matching dashboard behavior) unless --all-environments is set
	if !allEnvironments {
		registeredAgents, err = FilterAgentsByCurrentEnvironment(cmd, registeredAgents)
		ui.ExitOnError("filtering agents by environment", err)
	}

	agents, err := GetKubernetesAgents([]string{""})
	ui.ExitOnError("listing pods", err)

	agents = CombineAgents(agents, registeredAgents)

	// Filter agents based on criteria
	filteredAgents := make(internalAgents, 0, len(agents))

	if showDeleted {
		// ONLY show deleted agents
		for _, agent := range agents {
			if agent.Registered != nil && agent.Registered.DeletedAt != nil {
				filteredAgents = append(filteredAgents, agent)
			}
		}
	} else if showUnknown {
		// ONLY show unknown agents
		for _, agent := range agents {
			if agent.Registered == nil {
				filteredAgents = append(filteredAgents, agent)
			}
		}
	} else {
		// Default: show only active registered agents
		for _, agent := range agents {
			// Exclude deleted agents
			if agent.Registered != nil && agent.Registered.DeletedAt != nil {
				continue
			}
			// Exclude unknown agents
			if agent.Registered == nil {
				continue
			}
			filteredAgents = append(filteredAgents, agent)
		}
	}

	agents = filteredAgents

	if len(agents) == 0 {
		fmt.Println(ui.LightGray("\nNo agents found"))
		return
	}

	// Render based on output format
	outputType := cmd.Flag("output").Value.String()

	// For JSON/YAML/go-template output, serialize the actual agent data
	if outputType == "json" || outputType == "yaml" || outputType == "go" {
		agentList := agents.ToAgentList()
		err = render.List(cmd, agentList, os.Stdout)
		ui.PrintOnError("Rendering list", err)
		return
	}

	// For pretty output, use table formatting
	var tableData ui.TableData
	if showUnknown {
		tableData = unknownAgentsTable{agents: agents}
	} else {
		tableData = agentsTableWithEnv{agents: agents, showEnvironments: allEnvironments}
	}

	err = render.List(cmd, tableData, os.Stdout)
	ui.PrintOnError("Rendering list", err)
}
