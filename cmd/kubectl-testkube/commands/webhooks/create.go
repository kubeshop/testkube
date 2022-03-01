package webhooks

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiClient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateWebhookCmd() *cobra.Command {
	var (
		types                         []string
		name, webhookType, image, uri string
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

			options := apiClient.CreateWebhookOptions{
				Name:        name,
				Namespace:   namespace,
				Types:       types,
				WebhookType: webhookType,
				Image:       image,
				Uri:         uri,
			}

			_, err = client.CreateWebhook(options)
			ui.ExitOnError("creating webhook "+name+" in namespace "+namespace, err)

			ui.Success("Webhook created", name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique test name - mandatory")
	cmd.Flags().StringArrayVarP(&types, "types", "t", []string{}, "types handled by exeutor")
	cmd.Flags().StringVar(&webhookType, "webhook-type", "job", "webhook type (defaults to job)")

	cmd.Flags().StringVarP(&uri, "uri", "u", "", "if resource need to be loaded from URI")
	cmd.Flags().StringVarP(&image, "image", "i", "", "if uri is git repository we can set additional branch parameter")

	return cmd
}
