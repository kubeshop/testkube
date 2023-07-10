package webhooks

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

func NewGetWebhookCmd() *cobra.Command {
	var name string
	var selectors []string
	var crdOnly bool

	cmd := &cobra.Command{
		Use:     "webhook <webhookName>",
		Aliases: []string{"webhooks", "wh"},
		Short:   "Get webhook details",
		Long:    `Get webhook, you can change output format, to get single details pass name as first arg`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			firstEntry := true
			if len(args) > 0 {
				name := args[0]
				webhook, err := client.GetWebhook(name)
				ui.ExitOnError("getting webhook: "+name, err)

				if crdOnly {
					if webhook.PayloadTemplate != "" {
						webhook.PayloadTemplate = fmt.Sprintf("%q", webhook.PayloadTemplate)
					}

					common.UIPrintCRD(crd.TemplateWebhook, webhook, &firstEntry)
					return
				}

				err = render.Obj(cmd, webhook, os.Stdout)
				ui.ExitOnError("rendering obj", err)
			} else {
				webhooks, err := client.ListWebhooks(strings.Join(selectors, ","))
				ui.ExitOnError("getting webhooks", err)

				if crdOnly {
					for _, webhook := range webhooks {
						if webhook.PayloadTemplate != "" {
							webhook.PayloadTemplate = fmt.Sprintf("%q", webhook.PayloadTemplate)
						}

						common.UIPrintCRD(crd.TemplateWebhook, webhook, &firstEntry)
					}

					return
				}

				err = render.List(cmd, webhooks, os.Stdout)
				ui.ExitOnError("rendering list", err)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name, you can also pass it as argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test crd")

	return cmd
}
