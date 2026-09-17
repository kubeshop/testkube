package webhooks

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/crd"
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
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrAPIClientInitFailed,
					"Error creating the Testkube API client",
					common.APIClientHint,
					err,
				))
			}

			firstEntry := true
			if len(args) > 0 {
				name := args[0]
				webhook, err := client.GetWebhook(name)
				if err != nil {
					// The API answered that the webhook is absent, which is a different thing to tell the
					// user than a read that did not complete.
					if apiclient.IsNotFound(err) {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrResourceNotFound,
							"Webhook not found",
							"Check the webhook name, or list the webhooks with `testkube get webhooks`",
							err,
						))
					}
					common.HandleCLIError(common.NewCLIError(
						common.TKErrAPIReadFailed,
						"Error getting the webhook",
						common.APIReadHint,
						err,
					))
				}

				if crdOnly {
					webhook.QuoteTextFields()
					common.UIPrintCRD(crd.TemplateWebhook, webhook, &firstEntry)
					return
				}

				err = render.Obj(cmd, webhook, os.Stdout)
				if err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrOutputRenderFailed,
						"Error rendering the webhook",
						common.OutputRenderHint,
						err,
					))
				}
			} else {
				webhooks, err := client.ListWebhooks(strings.Join(selectors, ","))
				if err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrAPIReadFailed,
						"Error getting the webhooks",
						common.APIReadHint,
						err,
					))
				}

				if crdOnly {
					for _, webhook := range webhooks {
						webhook.QuoteTextFields()
						common.UIPrintCRD(crd.TemplateWebhook, webhook, &firstEntry)
					}

					return
				}

				err = render.List(cmd, webhooks, os.Stdout)
				if err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrOutputRenderFailed,
						"Error rendering the webhooks",
						common.OutputRenderHint,
						err,
					))
				}
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name, you can also pass it as argument")
	cmd.Flags().MarkShorthandDeprecated("name", "please use --name instead")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().BoolVar(&crdOnly, "crd-only", false, "show only test crd")

	return cmd
}
