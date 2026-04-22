package testtriggers

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateTestTriggerCmd() *cobra.Command {
	var file string
	var update bool

	cmd := &cobra.Command{
		Use:     "testtrigger",
		Aliases: []string{"testtriggers", "tt"},
		Short:   "Create a TestTrigger from a YAML/JSON manifest",
		Long:    `Create a TestTrigger. The manifest may be passed via --file or piped on stdin. Accepts both the flat REST shape (TestTriggerUpsertRequest) and the CRD shape (with spec:).`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			req, err := readTestTriggerFromInput(file)
			ui.ExitOnError("reading test trigger input", err)

			if update {
				existing, getErr := client.GetTestTrigger(req.Name)
				if getErr == nil && existing.Name != "" {
					updated, err := client.UpdateTestTrigger(apiv1.UpdateTestTriggerOptions(req))
					ui.ExitOnError("updating test trigger", err)
					err = render.Obj(cmd, updated, os.Stdout)
					ui.ExitOnError("rendering obj", err)
					return
				}
			}

			created, err := client.CreateTestTrigger(apiv1.CreateTestTriggerOptions(req))
			ui.ExitOnError("creating test trigger", err)

			err = render.Obj(cmd, created, os.Stdout)
			ui.ExitOnError("rendering obj", err)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "path to YAML/JSON manifest, or '-' / empty for stdin")
	cmd.Flags().BoolVar(&update, "update", false, "update if exists instead of failing")

	return cmd
}
