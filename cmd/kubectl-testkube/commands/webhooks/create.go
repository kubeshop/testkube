package webhooks

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/crd"
	webhooksmapper "github.com/kubeshop/testkube/pkg/mapper/webhooks"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateWebhookCmd() *cobra.Command {
	var (
		events             []string
		name, uri          string
		selector           string
		labels             map[string]string
		payloadObjectField string
		payloadTemplate    string
		headers            map[string]string
	)

	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"wh"},
		Short:   "Create new Webhook",
		Long:    `Create new Webhook Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
			ui.ExitOnError("parsing flag value", err)

			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			namespace := cmd.Flag("namespace").Value.String()
			payloadTemplate = cmd.Flag("payload-template").Value.String()
			payloadTemplateContent := ""
			if payloadTemplate != "" {
				b, err := os.ReadFile(payloadTemplate)
				ui.ExitOnError("reading job template", err)
				payloadTemplateContent = string(b)
			}

			var client apiv1.Client
			if !crdOnly {
				client, namespace, err = common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				webhook, _ := client.GetWebhook(name)
				if name == webhook.Name {
					ui.Failf("Webhook with name '%s' already exists in namespace %s", name, namespace)
				}
			}

			options := apiv1.CreateWebhookOptions{
				Name:               name,
				Namespace:          namespace,
				Events:             webhooksmapper.MapStringArrayToCRDEvents(events),
				Uri:                uri,
				Selector:           selector,
				Labels:             labels,
				PayloadObjectField: payloadObjectField,
				PayloadTemplate:    payloadTemplateContent,
				Headers:            headers,
			}

			if !crdOnly {
				_, err := client.CreateWebhook(options)
				ui.ExitOnError("creating webhook "+name+" in namespace "+namespace, err)

				ui.Success("Webhook created", name)
			} else {
				if options.PayloadTemplate != "" {
					options.PayloadTemplate = fmt.Sprintf("%q", options.PayloadTemplate)
				}

				data, err := crd.ExecuteTemplate(crd.TemplateWebhook, options)
				ui.ExitOnError("executing crd template", err)

				ui.Info(data)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name - mandatory")
	cmd.Flags().StringArrayVarP(&events, "events", "e", []string{}, "event types handled by executor e.g. start-test|end-test")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs")
	cmd.Flags().StringVarP(&selector, "selector", "", "", "expression to select tests and test suites for webhook events: --selector app=backend")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&payloadObjectField, "payload-field", "", "", "field to use for notification object payload")
	cmd.Flags().StringVarP(&payloadTemplate, "payload-template", "", "", "if webhook needs to send a custom notification, then a path to template file should be provided")
	cmd.Flags().StringToStringVarP(&headers, "header", "", nil, "webhook header value pair: --header Content-Type=application/xml")

	return cmd
}
