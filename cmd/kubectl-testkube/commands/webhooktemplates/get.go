package webhooktemplates

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetWebhookTemplateCmd() *cobra.Command {
	var name string
	var selectors []string
	var crdOnly bool

	cmd := &cobra.Command{
		Use:     "webhooktemplate <webhookTemplateName>",
		Aliases: []string{"webhooktemplates", "wht"},
		Short:   "Get webhook template details",
		Long:    `Get webhook template, you can change output format, to get single details pass name as first arg`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			firstEntry := true
			if len(args) > 0 {
				name := args[0]
				webhookTemplate, err := client.GetWebhookTemplate(name)
				ui.ExitOnError("getting webhook template: "+name, err)

				if crdOnly {
					webhookTemplate.QuoteTextFields()
					common.UIPrintCRD(crd.TemplateWebhookTemplate, webhookTemplate, &firstEntry)
					return
				}

				err = render.Obj(cmd, webhookTemplate, os.Stdout)
				ui.ExitOnError("rendering obj", err)
			} else {
				webhookTemplates, err := client.ListWebhookTemplates(strings.Join(selectors, ","))
				ui.ExitOnError("getting webhook templates", err)

				if crdOnly {
					for _, webhookTemplate := range webhookTemplates {
						webhookTemplate.QuoteTextFields()
						common.UIPrintCRD(crd.TemplateWebhookTemplate, webhookTemplate, &firstEntry)
					}

					return
				}

				err = render.List(cmd, webhookTemplates, os.Stdout)
				ui.ExitOnError("rendering list", err)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook template name, you can also pass it as argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test crd")

	return cmd
}
