package webhooks

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/spf13/cobra"
)

type WebhookRenderer interface {
	Render(result testkube.Webhook, writer io.Writer) error
}

type WebhookJSONRenderer struct {
}

func (r WebhookJSONRenderer) Render(result testkube.Webhook, writer io.Writer) error {
	return json.NewEncoder(writer).Encode(result)
}

type WebhookGoTemplateRenderer struct {
	Template string
}

func (r WebhookGoTemplateRenderer) Render(result testkube.Webhook, writer io.Writer) error {
	tmpl, err := template.New("result").Parse(r.Template)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, result)
}

type WebhookRawRenderer struct {
}

func (r WebhookRawRenderer) Render(webhook testkube.Webhook, writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Name: %s, URI: %s, Events: %v\n",
		webhook.Name,
		webhook.Uri,
		webhook.Events,
	)

	return err
}

func GetWebhookRenderer(cmd *cobra.Command) WebhookRenderer {
	output := cmd.Flag("output").Value.String()

	switch output {
	case OutputRAW:
		return WebhookRawRenderer{}
	case OutputJSON:
		return WebhookJSONRenderer{}
	case OutputGoTemplate:
		template := cmd.Flag("go-template").Value.String()
		return WebhookGoTemplateRenderer{Template: template}
	default:
		return WebhookRawRenderer{}
	}
}
