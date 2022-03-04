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
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create new Webhook",
		Long:    `Create new Webhook Custom Resource`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			var err error

			client, namespace := common.GetClient(cmd)

			webhook, _ := client.GetWebhook(namespace, name)
			if name == webhook.Name {
				ui.Failf("Webhook with name '%s' already exists in namespace %s", name, namespace)
			}

			options := apiv1.CreateWebhookOptions{
				Name:      name,
				Namespace: namespace,
				Events:    webhooksmapper.MapStringArrayToCRDEvents(events),
				Uri:       uri,
			}

			_, err = client.CreateWebhook(options)
			ui.ExitOnError("creating webhook "+name+" in namespace "+namespace, err)

			ui.Success("Webhook created", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique webhook name - mandatory")
	cmd.Flags().StringArrayVarP(&events, "events", "e", []string{}, "event types handled by executor e.g. start-test|end-test")
	cmd.Flags().StringVarP(&uri, "uri", "u", "", "URI which should be called when given event occurs")

	return cmd
}
