package testtriggers

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetTestTriggerCmd() *cobra.Command {
	var selectors []string

	cmd := &cobra.Command{
		Use:     "testtrigger <name>",
		Aliases: []string{"testtriggers", "tt"},
		Short:   "Get TestTrigger details",
		Long:    `Get a single TestTrigger by name, or list all matching ones. Use --label to filter.`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name := args[0]
				trigger, err := client.GetTestTrigger(name)
				ui.ExitOnError("getting test trigger: "+name, err)

				err = render.Obj(cmd, trigger, os.Stdout)
				ui.ExitOnError("rendering obj", err)
				return
			}

			triggers, err := client.ListTestTriggers(strings.Join(selectors, ","))
			ui.ExitOnError("listing test triggers", err)

			err = render.List(cmd, testkube.TestTriggers(triggers), os.Stdout)
			ui.ExitOnError("rendering list", err)
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label selector, e.g. --label app=api")

	return cmd
}
