package webhooks

import (
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListWebhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Gets webhooks",
		Long:  `Gets webhook, you can change output format`,
		Run: func(cmd *cobra.Command, args []string) {

			client, _ := common.GetClient(cmd)
			webhooks, err := client.ListWebhooks()
			ui.ExitOnError("listing webhooks: ", err)

			render := GetWebhookListRenderer(cmd)
			err = render.Render(webhooks, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}

	return cmd
}
