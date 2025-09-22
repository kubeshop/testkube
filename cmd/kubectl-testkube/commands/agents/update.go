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
	)

	cmd := &cobra.Command{
		Use:  "agent <name>",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			UiUpdateAgent(cmd, strings.Join(args, ""), setLabels, deleteLabels)
		},
	}

	cmd.Flags().StringSliceVarP(&setLabels, "label", "l", nil, "labels to set")
	cmd.Flags().StringSliceVarP(&deleteLabels, "delete-label", "L", nil, "labels to delete")

	return cmd
}

func UiUpdateAgent(cmd *cobra.Command, name string, setLabels, deleteLabels []string) {
	agent, err := GetControlPlaneAgent(cmd, name)
	ui.ExitOnError("getting agent", err)

	if agent.Labels == nil {
		agent.Labels = make(map[string]string)
	}

	for _, labelPair := range setLabels {
		k, v, _ := strings.Cut(labelPair, "=")
		if v == "" {
			delete(agent.Labels, k)
		} else {
			agent.Labels[k] = v
		}
	}

	for _, labelName := range deleteLabels {
		delete(agent.Labels, labelName)
	}

	agent, err = UpdateAgent(cmd, agent.ID, client.AgentInput{
		Labels: common.Ptr(agent.Labels),
	})
	ui.ExitOnError("updating agent", err)

	PrintControlPlaneAgent(*agent)
}
