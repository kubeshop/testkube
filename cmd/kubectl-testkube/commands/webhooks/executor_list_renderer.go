package webhooks

import (
	"encoding/json"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

type WebhookListRenderer interface {
	Render(list testkube.WebhooksDetails, writer io.Writer) error
}

type WebhookJSONListRenderer struct {
}

func (r WebhookJSONListRenderer) Render(list testkube.WebhooksDetails, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(list)
}

type WebhookGoTemplateListRenderer struct {
	Template string
}

func (r WebhookGoTemplateListRenderer) Render(list testkube.WebhooksDetails, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	for _, webhookDetails := range list {
		err := tmpl.Execute(writer, webhookDetails)
		if err != nil {
			return err
		}

	}

	return nil
}

type WebhookRawListRenderer struct {
}

func (r WebhookRawListRenderer) Render(list testkube.WebhooksDetails, writer io.Writer) error {
	ui.Table(list, writer)
	return nil
}

func GetWebhookListRenderer(cmd *cobra.Command) WebhookListRenderer {
	output := cmd.Flag("output").Value.String()

	switch output {
	case OutputRAW:
		return WebhookRawListRenderer{}
	case OutputJSON:
		return WebhookJSONListRenderer{}
	case OutputGoTemplate:
		template := cmd.Flag("go-template").Value.String()
		return WebhookGoTemplateListRenderer{Template: template}
	default:
		return WebhookRawListRenderer{}
	}
}
