package agents

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
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
		Short:   "Get agents registered in the current environment",
		Long:    `Get details of a specific agent or list all agents. By default only active agents in the current environment are shown. Use --all-environments to list across environments, --show-deleted to view deleted agents, or --show-unknown to find cluster agents not registered in the control plane.`,
		Aliases: []string{"agents", "a"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if allEnvironments && showUnknown {
				ui.Warn("Note: --all-environments is ignored when using --show-unknown (unknown agents have no environment registration)")
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
	if err != nil {
		common.HandleCLIError(common.NewCLIError(
			common.TKErrAgentGetFailed,
			"Error getting the agents",
			common.AgentLookupHint,
			err,
		))
	}

	namespaces, err := GetKubernetesNamespaces()
	if err != nil {
		common.HandleCLIError(common.NewCLIError(
			common.TKErrResourceLookupFailed,
			"Error listing the Kubernetes namespaces",
			common.ClusterLookupHint,
			err,
		))
	}

	agents, err := GetKubernetesAgents(namespaces)
	if err != nil {
		common.HandleCLIError(common.NewCLIError(
			common.TKErrResourceLookupFailed,
			"Error getting the agents running in the cluster",
			common.ClusterLookupHint,
			err,
		))
	}

	agents = CombineAgents(agents, registeredAgents)

	var agent *internalAgent
	for _, a := range agents {
		if a.AgentID.Value == agentId || (a.Registered != nil && (a.Registered.ID == agentId || a.Registered.Name == agentId)) {
			agent = &a
			break
		}
	}
	if agent == nil {
		common.HandleCLIError(common.NewCLIError(
			common.TKErrResourceNotFound,
			"Agent not found",
			"Check the agent name or ID, or list the agents with `testkube get agents`. Add --show-unknown to include cluster agents that are not registered, and --show-deleted to include deleted ones",
			fmt.Errorf("agent '%s' not found", agentId),
		))
	}

	if decryptSecretKey {
		secretKey, err := GetControlPlaneAgentSecretKey(cmd, agent.Registered.ID)
		if err != nil {
			common.HandleCLIError(common.NewCLIError(
				common.TKErrAgentGetFailed,
				"Error getting the decrypted agent secret key",
				"Check that your credentials are valid and that your user can read the secret key of this agent",
				err,
			))
		}
		agent.Registered.SecretKey = secretKey
	}

	PrintControlPlaneAgent(*agent.Registered)
}

func UiListAgents(cmd *cobra.Command, showUnknown bool, showDeleted bool, allEnvironments bool) {
	registeredAgents, err := GetControlPlaneAgents(cmd, showDeleted)
	if err != nil {
		// The hint of a lookup by name points at `testkube get agents`, which is this command.
		common.HandleCLIError(common.NewCLIError(
			common.TKErrAgentGetFailed,
			"Error getting the agents",
			"Check that your credentials are valid and that the current context points at the organization and environment you expect",
			err,
		))
	}

	// Filter agents by current environment (matching dashboard behavior) unless --all-environments is set
	if !allEnvironments {
		cfg, err := config.Load()
		if err != nil {
			common.HandleCLIError(common.NewCLIError(
				common.TKErrConfigInitFailed,
				"Error loading testkube config file",
				common.ConfigFileHint,
				err,
			))
		}
		registeredAgents = FilterAgentsByEnvironment(registeredAgents, cfg.CloudContext.EnvironmentId)
	}

	agents, err := GetKubernetesAgents([]string{""})
	if err != nil {
		common.HandleCLIError(common.NewCLIError(
			common.TKErrResourceLookupFailed,
			"Error getting the agents running in the cluster",
			common.ClusterLookupHint,
			err,
		))
	}

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
		ui.Print(ui.LightGray("\nNo agents found"))
		return
	}

	// Render based on output format
	outputType := cmd.Flag("output").Value.String()

	// For JSON/YAML/go-template output, serialize the actual agent data
	if outputType == "json" || outputType == "yaml" || outputType == "go" {
		if showUnknown {
			err = render.List(cmd, agents.ToUnknownAgentList(), os.Stdout)
		} else {
			err = render.List(cmd, agents.ToAgentList(), os.Stdout)
		}
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
