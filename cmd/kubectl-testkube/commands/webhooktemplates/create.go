package webhooktemplates

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/crd"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateWebhookTemplateCmd() *cobra.Command {
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
	)

	cmd := &cobra.Command{
		Use:     "webhooktemplate",
		Aliases: []string{"wh"},
		Short:   "Create new Webhook Template",
		Long:    `Create new Webhook Template Custom Resource`,
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

				webhookTemplate, _ := client.GetWebhookTemplate(name)
				if name == webhookTemplate.Name {
					if cmd.Flag("update").Changed {
						if !update {
							ui.Failf("Webhook Template with name '%s' already exists in namespace %s, ", webhookTemplate.Name, namespace)
						}
					} else {
						ok := ui.Confirm(fmt.Sprintf("Webhook Template with name '%s' already exists in namespace %s, ", webhookTemplate.Name, namespace) +
							"do you want to overwrite it?")
						if !ok {
							ui.Failf("Webhook Template creation was aborted")
						}
					}

					options, err := NewUpdateWebhookTemplateOptionsFromFlags(cmd)
					ui.ExitOnError("getting webhook template options", err)

					_, err = client.UpdateWebhookTemplate(options)
					ui.ExitOnError("updating webhook template "+name+" in namespace "+namespace, err)

					ui.SuccessAndExit("Webhook template updated", name)
				}
			}

			options, err := NewCreateWebhookTemplateOptionsFromFlags(cmd)
			ui.ExitOnError("getting webhook template options", err)

			if !crdOnly {
				_, err := client.CreateWebhookTemplate(options)
				ui.ExitOnError("creating webhook template "+name+" in namespace "+namespace, err)

				ui.Success("Webhook template created", name)
			} else {
				if options.PayloadTemplate != "" {
					options.PayloadTemplate = fmt.Sprintf("%q", options.PayloadTemplate)
				}

				data, err := crd.ExecuteTemplate(crd.TemplateWebhookTemplate, options)
				ui.ExitOnError("executing crd template", err)

				ui.Info(data)
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook template name - mandatory")
	cmd.Flags().StringArrayVarP(&events, "events", "e", []string{}, "event types handled by webhook template e.g. start-test|end-test")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs (golang template supported)")
	cmd.Flags().StringVarP(&selector, "selector", "", "", "expression to select tests, test suites, test workflows for webhook template events: --selector app=backend")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().StringVarP(&payloadObjectField, "payload-field", "", "", "field to use for notification object payload")
	cmd.Flags().StringVarP(&payloadTemplate, "payload-template", "", "", "if webhook template needs to send a custom notification, then a path to template file should be provided")
	cmd.Flags().StringToStringVarP(&headers, "header", "", nil, "webhook template header value pair (golang template supported): --header Content-Type=application/xml")
	cmd.Flags().StringVar(&payloadTemplateReference, "payload-template-reference", "", "reference to payload template to use for the webhook template")
	cmd.Flags().StringToStringVarP(&config, "config", "", nil, "webhook template config variable with csv coluums (value=data or secret=namespace;name;key): --config var1=\"value=data\" or --config var2=\"secret=ns1;name1;key1\"")
	cmd.Flags().StringToStringVarP(&parameters, "parameter", "", nil, "webhook template parameter variable with csv coluums (description;required;example;default;pattern): --parameter var3=\"descr;true;12345;0;[0-9]*\"")
	cmd.Flags().BoolVar(&update, "update", false, "update, if webhook template already exists")
	cmd.Flags().BoolVar(&disable, "disable", false, "disable webhook template")

	return cmd
}
