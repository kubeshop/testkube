package workflowtriggers

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewUpdateWorkflowTriggerCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:     "workflowtrigger",
		Aliases: []string{"workflowtriggers", "wt"},
		Short:   "Update an existing WorkflowTrigger (v2) from a YAML/JSON manifest",
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			trigger, err := readWorkflowTriggerFromInput(file)
			ui.ExitOnError("reading workflow trigger input", err)

			updated, err := client.UpdateWorkflowTrigger(trigger)
			ui.ExitOnError("updating workflow trigger", err)

			err = render.Obj(cmd, updated, os.Stdout)
			ui.ExitOnError("rendering obj", err)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "path to YAML/JSON manifest, or '-' / empty for stdin")

	return cmd
}
