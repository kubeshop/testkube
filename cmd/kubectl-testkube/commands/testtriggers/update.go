package testtriggers

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewUpdateTestTriggerCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:     "testtrigger",
		Aliases: []string{"testtriggers", "tt"},
		Short:   "Update an existing TestTrigger from a YAML/JSON manifest",
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			req, err := readTestTriggerFromInput(file)
			ui.ExitOnError("reading test trigger input", err)

			updated, err := client.UpdateTestTrigger(apiv1.UpdateTestTriggerOptions(req))
			ui.ExitOnError("updating test trigger", err)

			err = render.Obj(cmd, updated, os.Stdout)
			ui.ExitOnError("rendering obj", err)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "path to YAML/JSON manifest, or '-' / empty for stdin")

	return cmd
}
