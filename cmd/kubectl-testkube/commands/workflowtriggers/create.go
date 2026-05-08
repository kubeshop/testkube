package workflowtriggers

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateWorkflowTriggerCmd() *cobra.Command {
	var file string
	var update bool

	cmd := &cobra.Command{
		Use:     "workflowtrigger",
		Aliases: []string{"workflowtriggers", "wt"},
		Short:   "Create a WorkflowTrigger (v2) from a YAML/JSON manifest",
		Long:    `Create a WorkflowTrigger. The manifest may be passed via --file or piped on stdin. Accepts both the flat REST shape and the CRD shape.`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			trigger, err := readWorkflowTriggerFromInput(file)
			ui.ExitOnError("reading workflow trigger input", err)

			if update {
				// Upsert: update if exists, else create.
				existing, getErr := client.GetWorkflowTrigger(trigger.Name)
				if getErr == nil && existing.Name != "" {
					updated, err := client.UpdateWorkflowTrigger(trigger)
					ui.ExitOnError("updating workflow trigger", err)
					err = render.Obj(cmd, updated, os.Stdout)
					ui.ExitOnError("rendering obj", err)
					return
				}
			}

			created, err := client.CreateWorkflowTrigger(trigger)
			ui.ExitOnError("creating workflow trigger", err)

			err = render.Obj(cmd, created, os.Stdout)
			ui.ExitOnError("rendering obj", err)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "path to YAML/JSON manifest, or '-' / empty for stdin")
	cmd.Flags().BoolVar(&update, "update", false, "update if exists instead of failing")

	return cmd
}
