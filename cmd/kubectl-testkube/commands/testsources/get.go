package testsources

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testsources/renderer"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetTestSourceCmd() *cobra.Command {
	var name string
	var selectors []string
	var crdOnly bool

	cmd := &cobra.Command{
		Use:     "testsource <testSourceName>",
		Aliases: []string{"testsources", "tsc"},
		Short:   "Get test source details",
		Long:    `Get test source, you can change output format, to get single details pass name as first arg`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			firstEntry := true
			if len(args) > 0 {
				name := args[0]
				testSource, err := client.GetTestSource(name)
				ui.ExitOnError("getting test source: "+name, err)

				if crdOnly {
					if testSource.Data != "" {
						testSource.Data = fmt.Sprintf("%q", testSource.Data)
					}

					common.UIPrintCRD(crd.TemplateTestSource, testSource, &firstEntry)
					return
				}

				err = render.Obj(cmd, testSource, os.Stdout, renderer.TestSourceRenderer)
				ui.ExitOnError("rendering obj", err)
			} else {
				testSources, err := client.ListTestSources(strings.Join(selectors, ","))
				ui.ExitOnError("getting test sources", err)

				if crdOnly {
					for _, testSource := range testSources {
						if testSource.Data != "" {
							testSource.Data = fmt.Sprintf("%q", testSource.Data)
						}

						common.UIPrintCRD(crd.TemplateTestSource, testSource, &firstEntry)
					}

					return
				}

				err = render.List(cmd, testSources, os.Stdout)
				ui.ExitOnError("rendering list", err)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique test source name, you can also pass it as argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test crd")

	return cmd
}
