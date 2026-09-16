package webhooks

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
	)

	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"wh"},
		Short:   "Create new Webhook",
		Long:    `Create new Webhook Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			crdOnly, err := strconv.ParseBool(cmd.Flag("crd-only").Value.String())
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrInvalidRuntimeParameter,
					"Error reading the crd-only flag",
					common.BoolFlagValueHint,
					err,
				))
			}

			if name == "" {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrInvalidRuntimeParameter,
					"No webhook name provided",
					common.NameFlagHint,
					errors.New("no webhook name provided"),
				))
			}

			// The namespace comes back from GetClient; the flag is read there, not here.
			var (
				client    apiv1.Client
				namespace string
			)
			if !crdOnly {
				client, namespace, err = common.GetClient(cmd)
				if err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrAPIClientInitFailed,
						"Error creating the Testkube API client",
						common.APIClientHint,
						err,
					))
				}

				// A 404 means there is nothing to overwrite and creation carries on below. Any other
				// failure means the lookup never answered, so stop instead of silently creating.
				webhook, err := client.GetWebhook(name)
				if err != nil && !apiv1.IsNotFound(err) {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrAPIReadFailed,
						"Error checking whether the webhook already exists",
						common.APIReadHint,
						err,
					))
				}

				if err == nil && name == webhook.Name {
					if cmd.Flag("update").Changed {
						if !update {
							common.HandleCLIError(common.NewCLIError(
								common.TKErrInvalidRuntimeParameter,
								"Webhook already exists",
								common.NameConflictHint,
								fmt.Errorf("webhook '%s' already exists in namespace '%s'", webhook.Name, namespace),
							))
						}
					} else {
						ok := ui.Confirm(fmt.Sprintf("Webhook with name '%s' already exists in namespace %s, ", webhook.Name, namespace) +
							"do you want to overwrite it?")
						if !ok {
							ui.Failf("Webhook creation was aborted")
						}
					}

					options, err := NewUpdateWebhookOptionsFromFlags(cmd)
					if err != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrInvalidRuntimeParameter,
							"Error reading the webhook flags",
							common.WebhookFlagsHint,
							err,
						))
					}

					_, err = client.UpdateWebhook(options)
					if err != nil {
						common.HandleCLIError(common.NewCLIError(
							common.TKErrAPIWriteFailed,
							"Error updating the webhook",
							common.APIWriteHint,
							err,
						))
					}

					ui.SuccessAndExit("Webhook updated", name)
				}
			}

			options, err := NewCreateWebhookOptionsFromFlags(cmd)
			if err != nil {
				common.HandleCLIError(common.NewCLIError(
					common.TKErrInvalidRuntimeParameter,
					"Error reading the webhook flags",
					common.WebhookFlagsHint,
					err,
				))
			}

			if !crdOnly {
				_, err := client.CreateWebhook(options)
				if err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrAPIWriteFailed,
						"Error creating the webhook",
						common.APIWriteHint,
						err,
					))
				}

				ui.Success("Webhook created", name)
			} else {
				(*testkube.WebhookCreateRequest)(&options).QuoteTextFields()

				data, err := crd.ExecuteTemplate(crd.TemplateWebhook, options)
				if err != nil {
					common.HandleCLIError(common.NewCLIError(
						common.TKErrOutputRenderFailed,
						"Error rendering the webhook CRD",
						"Check the flag values that go into the CRD, or drop '--crd-only' to create the webhook through the Testkube API",
						err,
					))
				}

				fmt.Print(data)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name - mandatory")
	cmd.Flags().MarkShorthandDeprecated("name", "please use --name instead")
	cmd.Flags().StringArrayVarP(&events, "events", "e", []string{}, "event types handled by webhook e.g. start-test|end-test")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs (golang template supported)")
	cmd.Flags().StringVarP(&selector, "selector", "", "", "expression to select tests, test suites, test workflows for webhook events: --selector app=backend")
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
	cmd.Flags().MarkDeprecated("enable", "enable webhook is deprecated")

	return cmd
}
