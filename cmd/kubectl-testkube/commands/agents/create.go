package agents

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/internal/common"
	cloudclient "github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateAgentCommand() *cobra.Command {
	var (
		agentType      string
		labelPairs     []string
		environmentIds []string
	)
	cmd := &cobra.Command{
		Use:  "agent",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agent := UiCreateAgent(cmd, agentType, args[0], labelPairs, environmentIds, false, "", false)
			cliType, _ := GetCliAgentType(agent.Type)
			ui.NL()
			ui.Info("Install the agent with command:")
			ui.ShellCommand(
				fmt.Sprintf("kubectl testkube install %s %s --secret %s", cliType, agent.Name, agent.SecretKey),
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
		global         bool
		group          string
		floating       bool
	)
	cmd := &cobra.Command{
		Use:  "runner",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agent := UiCreateAgent(cmd, "runner", args[0], labelPairs, environmentIds, global, group, floating)
			cliType, _ := GetCliAgentType(agent.Type)
			ui.NL()
			ui.Info("Install the agent with command:")
			ui.ShellCommand(
				fmt.Sprintf("kubectl testkube install %s %s --secret %s", cliType, agent.Name, agent.SecretKey),
			)
		},
	}

	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&global, "global", false, "make it global runner")
	cmd.Flags().StringVar(&group, "group", "", "make it grouped runner")
	cmd.Flags().BoolVar(&floating, "floating", false, "create as a floating runner")

	return cmd
}

func NewCreateGitOpsCommand() *cobra.Command {
	var (
		labelPairs     []string
		environmentIds []string
	)
	cmd := &cobra.Command{
		Use:  "gitops",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agent := UiCreateAgent(cmd, "gitops", args[0], labelPairs, environmentIds, false, "", false)
			cliType, _ := GetCliAgentType(agent.Type)
			ui.NL()
			ui.Info("Install the agent with command:")
			ui.ShellCommand(
				fmt.Sprintf("kubectl testkube install %s %s --secret %s", cliType, agent.Name, agent.SecretKey),
			)
		},
	}

	cmd.Flags().StringSliceVarP(&environmentIds, "env", "e", nil, "environment ID or slug that the agent have access to")
	cmd.Flags().StringSliceVarP(&labelPairs, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}

func UiCreateAgent(cmd *cobra.Command, agentType string, name string, labelPairs []string, environmentIds []string, isGlobalRunner bool, runnerGroup string, floating bool) *cloudclient.Agent {
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
		Labels:       common.Ptr(make(map[string]string)),
		Environments: environmentIds,
		Floating:     floating,
	}

	if runnerGroup != "" {
		(*input.Labels)["group"] = runnerGroup
		input.RunnerPolicy = &cloudclient.RunnerPolicy{
			RequiredMatch: []string{"group"},
		}
	} else if !isGlobalRunner {
		input.RunnerPolicy = &cloudclient.RunnerPolicy{
			RequiredMatch: []string{"name"},
		}
	}

	input.Type, _ = GetInternalAgentType(agentType)

	for _, label := range labelPairs {
		k, v, _ := strings.Cut(label, "=")
		(*input.Labels)[k] = v
	}

	envs, err := GetControlPlaneEnvironments(cmd)
	ui.ExitOnError("getting environments", err)

	if len(input.Environments) == 0 {
		cfg, err := config.Load()
		ui.ExitOnError("loading config", err)
		envOpts := []string{envs[cfg.CloudContext.EnvironmentId].Slug}
		for id := range envs {
			if id != cfg.CloudContext.EnvironmentId {
				envOpts = append(envOpts, id)
			}
		}
		input.Environments = []string{ui.Select("select environment", envOpts)}
	}

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

	// Validate if the environments have the next architecture enabled
	for _, envId := range input.Environments {
		env, ok := envs[envId]
		if !ok {
			ui.Failf("unknown environment: %s", envId)
		}
		if !env.NewArchitecture {
			ui.Warn(fmt.Sprintf("Environment '%s' (%s) does not support new architecture.", env.Name, env.Id))
			if !ui.Confirm("do you want to enable it?") {
				os.Exit(1)
			}
			err := EnableNewArchitecture(cmd, env)
			ui.ExitOnError("enabling new architecture", err)
		}
	}

	agent, err := CreateAgent(cmd, input)
	ui.ExitOnError("creating agent", err)

	PrintControlPlaneAgent(*agent)

	return agent
}
