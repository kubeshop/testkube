package agents

import (
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
	cmd.Flags().StringVarP(&runnerMode, "group-name", "g", "", "group name used to target executions")

	return cmd
}

func UiUpdateAgent(cmd *cobra.Command, name string, setLabels, deleteLabels []string, runnerMode string, groupName string) {
	agent, err := GetControlPlaneAgent(cmd, name)
	ui.ExitOnError("getting agent", err)

	if agent.Labels == nil {
		agent.Labels = make(map[string]string)
	}

	if runnerMode == "group" && groupName != "" && agent.Labels["group"] == "" {
		ui.ExitOnError("group name is mandatory to set runner mode as group, use param --group-name")
	}

	if runnerMode != "" {
		if runnerMode == "group" || runnerMode == "global" || runnerMode == "independent" {
			if runnerMode == "independent" {
				runnerMode = "name"
			}
			agent.RunnerPolicy = &client.RunnerPolicy{
				RequiredMatch: []string{runnerMode},
			}
		} else {
			ui.ExitOnError("runner mode not valid, allowed values:  independent, group, global")
		}
	}

	for _, labelPair := range setLabels {
		k, v, _ := strings.Cut(labelPair, "=")
		if v == "" {
			if agent.RunnerPolicy.RequiredMatch[0] == "group" && k == "group" {
				ui.ExitOnError("cannot delete group label when runner mode is group")
			}
			delete(agent.Labels, k)
		} else {
			agent.Labels[k] = v
		}
	}

	for _, labelName := range deleteLabels {
		if agent.RunnerPolicy.RequiredMatch[0] == "group" && labelName == "group" {
			ui.ExitOnError("cannot delete group label when runner mode is group")
		}
		delete(agent.Labels, labelName)
	}

	agent, err = UpdateAgent(cmd, agent.ID, client.AgentInput{
		RunnerPolicy: common.Ptr(*agent.RunnerPolicy),
		Labels: common.Ptr(agent.Labels),
	})
	ui.ExitOnError("updating agent", err)

	PrintControlPlaneAgent(*agent)
}
