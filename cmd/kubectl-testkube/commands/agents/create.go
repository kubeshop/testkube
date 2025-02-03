package agents

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateAgentCommand() *cobra.Command {
	var (
		agentType      string
		labelPairs     []string
		environmentIds []string
	)
	cmd := &cobra.Command{
		Use:    "agent",
		Args:   cobra.ExactArgs(1),
		Hidden: !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			agent := UiCreateAgent(cmd, agentType, args[0], labelPairs, environmentIds)
			ui.NL()
			ui.Info("Install the agent with command:")
			ui.ShellCommand(
				fmt.Sprintf("kubectl testkube install %s %s --secret %s --wizard", agent.Type, agent.Name, agent.SecretKey),
			)
		},
	}

	cmd.Flags().StringVarP(&agentType, "type", "t", "", "agent type, one of: runner, gitops")
	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}

func NewCreateRunnerCommand() *cobra.Command {
	var (
		labelPairs     []string
		environmentIds []string
	)
	cmd := &cobra.Command{
		Use:    "runner",
		Args:   cobra.ExactArgs(1),
		Hidden: !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			agent := UiCreateAgent(cmd, "runner", args[0], labelPairs, environmentIds)
			ui.NL()
			ui.Info("Install the agent with command:")
			ui.ShellCommand(
				fmt.Sprintf("kubectl testkube install %s %s --secret %s --wizard", agent.Type, agent.Name, agent.SecretKey),
			)
		},
	}

	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}

func NewCreateGitOpsCommand() *cobra.Command {
	var (
		labelPairs     []string
		environmentIds []string
	)
	cmd := &cobra.Command{
		Use:    "gitops",
		Args:   cobra.ExactArgs(1),
		Hidden: !log.IsTrue("EXPERIMENTAL"),
		Run: func(cmd *cobra.Command, args []string) {
			agent := UiCreateAgent(cmd, "gitops", args[0], labelPairs, environmentIds)
			ui.NL()
			ui.Info("Install the agent with command:")
			ui.ShellCommand(
				fmt.Sprintf("kubectl testkube install %s %s --secret %s --wizard", agent.Type, agent.Name, agent.SecretKey),
			)
		},
	}

	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}

func UiCreateAgent(cmd *cobra.Command, agentType string, name string, labelPairs []string, environmentIds []string) *cloudclient.Agent {
	if name == "" {
		name = ui.TextInput("agent name")
		if name == "" {
			ui.Failf("agent name is required")
		}
	}

	// Get existing agent of that name
	if existing, err := GetControlPlaneAgent(cmd, name); err == nil {
		ui.Failf("agent '%s' already exists", existing.Name)
	}

	if agentType == "" {
		agentType = ui.Select("select type", []string{
			"runner",
			"gitops",
		})
		if name == "" {
			ui.Failf("agent type is required")
		}
	}

	input := cloudclient.AgentInput{
		Name:         name,
		Labels:       make(map[string]string),
		Environments: environmentIds,
	}

	input.Type, _ = GetInternalAgentType(agentType)

	for _, label := range labelPairs {
		k, v, _ := strings.Cut(label, "=")
		input.Labels[k] = v
	}

	envs, err := GetControlPlaneEnvironments(cmd)
	ui.ExitOnError("getting environments", err)
	for i, envId := range input.Environments {
		_, ok := envs[envId]
		if !ok {
			for _, env := range envs {
				if env.Slug == envId {
					input.Environments[i] = env.Id
					break
				}
			}
		}
	}

	if len(input.Environments) == 0 {
		cfg, err := config.Load()
		ui.ExitOnError("loading config", err)
		input.Environments = []string{cfg.CloudContext.EnvironmentId}
	}

	agent, err := CreateAgent(cmd, input)
	ui.ExitOnError("creating agent", err)

	PrintControlPlaneAgent(*agent)

	return agent
}
