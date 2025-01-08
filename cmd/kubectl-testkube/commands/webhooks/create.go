package webhooks

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateWebhookCmd() *cobra.Command {
	var (
		events                   []string
		name, uri                string
		selector                 string
		labels                   map[string]string
		payloadObjectField       string
		payloadTemplate          string
		headers                  map[string]string
		payloadTemplateReference string
		update                   bool
		disable                  bool
		config                   map[string]string
		parameters               map[string]string
		webhookTemplateReference string
		isTemplate               bool
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
			var client apiv1.Client
			if !crdOnly {
				client, namespace, err = common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				webhook, _ := client.GetWebhook(name)
				if name == webhook.Name {
					if cmd.Flag("update").Changed {
						if !update {
							ui.Failf("Webhook with name '%s' already exists in namespace %s, ", webhook.Name, namespace)
						}
					} else {
						ok := ui.Confirm(fmt.Sprintf("Webhook with name '%s' already exists in namespace %s, ", webhook.Name, namespace) +
							"do you want to overwrite it?")
						if !ok {
							ui.Failf("Webhook creation was aborted")
						}
					}

					options, err := NewUpdateWebhookOptionsFromFlags(cmd)
					ui.ExitOnError("getting webhook options", err)

					_, err = client.UpdateWebhook(options)
					ui.ExitOnError("updating webhook "+name+" in namespace "+namespace, err)

					ui.SuccessAndExit("Webhook updated", name)
				}
			}

			options, err := NewCreateWebhookOptionsFromFlags(cmd)
			ui.ExitOnError("getting webhook options", err)

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
	cmd.Flags().StringArrayVarP(&events, "events", "e", []string{}, "event types handled by webhook e.g. start-test|end-test")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs (golang template supported)")
	cmd.Flags().StringVarP(&selector, "selector", "", "", "expression to select tests and test suites for webhook events: --selector app=backend")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&payloadObjectField, "payload-field", "", "", "field to use for notification object payload")
	cmd.Flags().StringVarP(&payloadTemplate, "payload-template", "", "", "if webhook needs to send a custom notification, then a path to template file should be provided")
	cmd.Flags().StringToStringVarP(&headers, "header", "", nil, "webhook header value pair (golang template supported): --header Content-Type=application/xml")
	cmd.Flags().StringVar(&payloadTemplateReference, "payload-template-reference", "", "reference to payload template to use for the webhook")
	cmd.Flags().StringToStringVarP(&config, "config", "", nil, "webhook config variable with csv coluums (value=data or secret=namespace;name;key): --config var1=\"value=data\" or --config var2=\"secret=ns1;name1;key1\"")
	cmd.Flags().StringToStringVarP(&parameters, "parameter", "", nil, "webhook parameter variable with csv coluums (description;required;example;default;pattern): --parameter var3=\"descr;true;12345;0;[0-9]*\"")
	cmd.Flags().StringVar(&webhookTemplateReference, "webhook-template-reference", "", "reference to webhook to use as template for the webhook")
	cmd.Flags().BoolVar(&update, "update", false, "update, if webhook already exists")
	cmd.Flags().BoolVar(&disable, "disable", false, "disable webhook")
	cmd.Flags().BoolVar(&isTemplate, "is-template", false, "use webhook as a template")
	cmd.Flags().MarkDeprecated("enable", "enable webhook is deprecated")

	return cmd
}
