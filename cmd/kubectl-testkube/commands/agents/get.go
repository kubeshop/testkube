package agents

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetAgentCommand() *cobra.Command {
	var (
		decryptSecretKey bool
	)
	cmd := &cobra.Command{
		Args:    cobra.MaximumNArgs(1),
		Use:     "agent [name]",
		Aliases: []string{"agents", "a"},
		Hidden:  !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				UiListAgents(cmd, "")
			} else {
				UiGetAgent(cmd, args[0], decryptSecretKey)
			}
		},
	}

	cmd.Flags().BoolVar(&decryptSecretKey, "decrypted-secret", false, "should it fetch decrypted secret key")

	return cmd
}

func NewGetRunnerCommand() *cobra.Command {
	var (
		decryptSecretKey bool
	)
	cmd := &cobra.Command{
		Args:    cobra.MaximumNArgs(1),
		Use:     "runner [name]",
		Aliases: []string{"runners"},
		Hidden:  !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				UiListAgents(cmd, "runner")
			} else {
				UiGetAgent(cmd, args[0], decryptSecretKey)
			}
		},
	}

	cmd.Flags().BoolVar(&decryptSecretKey, "decrypted-secret", false, "should it fetch decrypted secret key")

	return cmd
}

func NewGetGitOpsCommand() *cobra.Command {
	var (
		decryptSecretKey bool
	)
	cmd := &cobra.Command{
		Args:    cobra.MaximumNArgs(1),
		Use:     "gitops [name]",
		Aliases: []string{},
		Hidden:  !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				UiListAgents(cmd, "gitops")
			} else {
				UiGetAgent(cmd, args[0], decryptSecretKey)
			}
		},
	}

	cmd.Flags().BoolVar(&decryptSecretKey, "decrypted-secret", false, "should it fetch decrypted secret key")

	return cmd
}

func UiGetAgent(cmd *cobra.Command, agentId string, decryptSecretKey bool) {
	registeredAgents, err := GetControlPlaneAgents(cmd, "")
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

func UiListAgents(cmd *cobra.Command, agentType string) {
	agentType, _ = GetInternalAgentType(agentType)

	registeredAgents, err := GetControlPlaneAgents(cmd, "")
	ui.ExitOnError("getting agents", err)

	agents, err := GetKubernetesAgents([]string{""})
	ui.ExitOnError("listing pods", err)

	agents = CombineAgents(agents, registeredAgents)

	var recognizedAgents, externalAgents, unknownAgents internalAgents
	for i := range agents {
		if agentType != "" && agents[i].Registered != nil && agents[i].Registered.Type != agentType {
			continue
		}
		if agents[i].Registered != nil && agents[i].Pod.Name != "" {
			recognizedAgents = append(recognizedAgents, agents[i])
		} else if agents[i].Registered != nil {
			externalAgents = append(externalAgents, agents[i])
		} else {
			unknownAgents = append(unknownAgents, agents[i])
		}
	}

	fmt.Println(color.Bold.Render("\nRecognized agents in current cluster"))
	if len(recognizedAgents) > 0 {
		ui.Table(recognizedAgents, os.Stdout)
	} else {
		fmt.Println(ui.LightGray("None"))
	}

	fmt.Println(color.Bold.Render("\nAgents outside of current cluster"))
	if len(externalAgents) > 0 {
		ui.Table(externalAgents, os.Stdout)
	} else {
		fmt.Println(ui.LightGray("None"))
	}

	if len(unknownAgents) > 0 {
		fmt.Println(color.Bold.Render("\nUnknown agents in current cluster"))
		ui.Table(unknownAgents, os.Stdout)
	}
}
