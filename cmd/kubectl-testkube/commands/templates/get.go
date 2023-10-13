package templates

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetTemplateCmd() *cobra.Command {
	var name string
	var selectors []string
	var crdOnly bool

	cmd := &cobra.Command{
		Use:     "template <templateName>",
		Aliases: []string{"templates", "tp"},
		Short:   "Get template details.",
		Long:    `Get template allows you to change the output format. To get single details, pass the template name as the first argument.`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			firstEntry := true
			if len(args) > 0 {
				name := args[0]
				template, err := client.GetTemplate(name)
				ui.ExitOnError("getting template: "+name, err)

				if crdOnly {
					if template.Body != "" {
						template.Body = fmt.Sprintf("%q", template.Body)
					}

					common.UIPrintCRD(crd.TemplateTemplate, template, &firstEntry)
					return
				}

				err = render.Obj(cmd, template, os.Stdout)
				ui.ExitOnError("rendering obj", err)
			} else {
				templates, err := client.ListTemplates(strings.Join(selectors, ","))
				ui.ExitOnError("getting templates", err)

				if crdOnly {
					for _, template := range templates {
						if template.Body != "" {
							template.Body = fmt.Sprintf("%q", template.Body)
						}

						common.UIPrintCRD(crd.TemplateTemplate, template, &firstEntry)
					}

					return
				}

				err = render.List(cmd, templates, os.Stdout)
				ui.ExitOnError("rendering list", err)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique template name, you can also pass it as argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test crd")

	return cmd
}
