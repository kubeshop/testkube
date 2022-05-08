package webhooks

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	webhooksmapper "github.com/kubeshop/testkube/pkg/mapper/webhooks"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateWebhookCmd() *cobra.Command {
	var (
		events    []string
		name, uri string
		labels    map[string]string
	)

	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"wh"},
		Short:   "Create new Webhook",
		Long:    `Create new Webhook Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {

			var err error

			client, namespace := common.GetClient(cmd)

			if name == "" {
				ui.Failf("pass valid name (in '--name' flag)")
			}

			webhook, _ := client.GetWebhook(name)
			if name == webhook.Name {
				ui.Failf("Webhook with name '%s' already exists in namespace %s", name, namespace)
			}

			options := apiv1.CreateWebhookOptions{
				Name:      name,
				Namespace: namespace,
				Events:    webhooksmapper.MapStringArrayToCRDEvents(events),
				Uri:       uri,
				Labels:    labels,
			}

			_, err = client.CreateWebhook(options)
			ui.ExitOnError("creating webhook "+name+" in namespace "+namespace, err)

			ui.Success("Webhook created", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name - mandatory")
	cmd.Flags().StringArrayVarP(&events, "events", "e", []string{}, "event types handled by executor e.g. start-test|end-test")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs")
	cmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
