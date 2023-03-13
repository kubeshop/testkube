package webhooks

import (
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
		events    []string
		name, uri string
		selector  string
		labels    map[string]string
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
				client, namespace = common.GetClient(cmd)

				webhook, _ := client.GetWebhook(name)
				if name == webhook.Name {
					ui.Failf("Webhook with name '%s' already exists in namespace %s", name, namespace)
				}
			}

			options := apiv1.CreateWebhookOptions{
				Name:      name,
				Namespace: namespace,
				Events:    webhooksmapper.MapStringArrayToCRDEvents(events),
				Uri:       uri,
				Selector:  selector,
				Labels:    labels,
			}

			if !crdOnly {
				_, err := client.CreateWebhook(options)
				ui.ExitOnError("creating webhook "+name+" in namespace "+namespace, err)

				ui.Success("Webhook created", name)
			} else {
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

	return cmd
}
