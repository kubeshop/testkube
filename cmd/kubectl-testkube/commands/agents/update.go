package agents

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/cloud/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewUpdateAgentCommand() *cobra.Command {
	var (
		setLabels    []string
		deleteLabels []string
		runnerMode   string
		groupName    string
	)

	cmd := &cobra.Command{
		Use:     "agent <name>",
		Aliases: []string{"runner"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UiUpdateAgent(cmd, strings.Join(args, ""), setLabels, deleteLabels, runnerMode, groupName)
		},
	}

	cmd.Flags().StringSliceVarP(&setLabels, "label", "l", nil, "labels to set")
	cmd.Flags().StringSliceVarP(&deleteLabels, "delete-label", "L", nil, "labels to delete")
	cmd.Flags().StringVarP(&runnerMode, "runner-mode", "m", "", "runner mode, allowed values: independent, group, global")
	cmd.Flags().StringVarP(&groupName, "group-name", "g", "", "group name used to target executions")

	return cmd
}

func UiUpdateAgent(cmd *cobra.Command, name string, setLabels, deleteLabels []string, runnerMode string, groupName string) {
	agent, err := GetControlPlaneAgent(cmd, name)
	ui.ExitOnError("getting agent", err)

	input, err := buildUpdateAgentInput(agent, setLabels, deleteLabels, runnerMode, groupName)
	ui.ExitOnError("preparing agent update", err)

	agent, err = UpdateAgent(cmd, agent.ID, input)
	ui.ExitOnError("updating agent", err)

	PrintControlPlaneAgent(*agent)
}

func buildUpdateAgentInput(agent *client.Agent, setLabels, deleteLabels []string, runnerMode string, groupName string) (client.AgentInput, error) {
	if agent.Labels == nil {
		agent.Labels = make(map[string]string)
	}

	if runnerMode != "" {
		switch runnerMode {
		case "independent":
			agent.RunnerPolicy = &client.RunnerPolicy{RequiredMatch: []string{"name"}}
		case "group":
			agent.RunnerPolicy = &client.RunnerPolicy{RequiredMatch: []string{"group"}}
			if groupName != "" {
				agent.Labels["group"] = groupName
			}
		case "global":
			agent.RunnerPolicy = &client.RunnerPolicy{RequiredMatch: []string{"global"}}
		default:
			return client.AgentInput{}, errors.New("runner mode not valid, allowed values: independent, group, global")
		}
	}

	for _, labelPair := range setLabels {
		k, v, _ := strings.Cut(labelPair, "=")
		if v == "" {
			if isAgentGroupRunner(agent) && k == "group" {
				return client.AgentInput{}, errors.New("cannot delete group label when runner mode is group")
			}
			delete(agent.Labels, k)
		} else {
			agent.Labels[k] = v
		}
	}

	for _, labelName := range deleteLabels {
		if isAgentGroupRunner(agent) && labelName == "group" {
			return client.AgentInput{}, errors.New("cannot delete group label when runner mode is group")
		}
		delete(agent.Labels, labelName)
	}

	if isAgentGroupRunner(agent) && agent.Labels["group"] == "" {
		return client.AgentInput{}, errors.New("group name is mandatory to set runner mode as group, use param --group-name")
	}

	input := client.AgentInput{
		Labels: common.Ptr(agent.Labels),
	}
	if agent.RunnerPolicy != nil {
		input.RunnerPolicy = common.Ptr(*agent.RunnerPolicy)
	}

	return input, nil
}

func isAgentGroupRunner(agent *client.Agent) bool {
	return agent != nil &&
		agent.RunnerPolicy != nil &&
		len(agent.RunnerPolicy.RequiredMatch) > 0 &&
		agent.RunnerPolicy.RequiredMatch[0] == "group"
}
