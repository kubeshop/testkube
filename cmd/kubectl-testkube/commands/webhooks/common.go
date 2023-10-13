package webhooks

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	webhooksmapper "github.com/kubeshop/testkube/pkg/mapper/webhooks"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewCreateWebhookOptionsFromFlags creates create webhook options from command flags
func NewCreateWebhookOptionsFromFlags(cmd *cobra.Command) (options apiv1.CreateWebhookOptions, err error) {
	name := cmd.Flag("name").Value.String()
	namespace := cmd.Flag("namespace").Value.String()
	events, err := cmd.Flags().GetStringArray("events")
	if err != nil {
		return options, err
	}

	payloadObjectField := cmd.Flag("payload-field").Value.String()
	payloadTemplate := cmd.Flag("payload-template").Value.String()
	payloadTemplateContent := ""
	if payloadTemplate != "" {
		b, err := os.ReadFile(payloadTemplate)
		ui.ExitOnError("reading payload template", err)
		payloadTemplateContent = string(b)
	}

	uri := cmd.Flag("uri").Value.String()
	selector := cmd.Flag("selector").Value.String()
	labels, err := cmd.Flags().GetStringToString("label")
	if err != nil {
		return options, err
	}

	headers, err := cmd.Flags().GetStringToString("header")
	if err != nil {
		return options, err
	}

	payloadTemplateReference := cmd.Flag("payload-template-reference").Value.String()
	options = apiv1.CreateWebhookOptions{
		Name:                     name,
		Namespace:                namespace,
		Events:                   webhooksmapper.MapStringArrayToCRDEvents(events),
		Uri:                      uri,
		Selector:                 selector,
		Labels:                   labels,
		PayloadObjectField:       payloadObjectField,
		PayloadTemplate:          payloadTemplateContent,
		Headers:                  headers,
		PayloadTemplateReference: payloadTemplateReference,
	}

	return options, nil
}

// NewUpdateWebhookOptionsFromFlags creates update webhook options from command flags
func NewUpdateWebhookOptionsFromFlags(cmd *cobra.Command) (options apiv1.UpdateWebhookOptions, err error) {
	var fields = []struct {
		name        string
		destination **string
	}{
		{
			"name",
			&options.Name,
		},
		{
			"uri",
			&options.Uri,
		},
		{
			"selector",
			&options.Selector,
		},
		{
			"payload-field",
			&options.PayloadObjectField,
		},
		{
			"payload-template-reference",
			&options.PayloadTemplateReference,
		},
	}

	for _, field := range fields {
		if cmd.Flag(field.name).Changed {
			value := cmd.Flag(field.name).Value.String()
			*field.destination = &value
		}
	}

	if cmd.Flag("payload-template").Changed {
		payloadTemplate := cmd.Flag("payload-template").Value.String()
		b, err := os.ReadFile(payloadTemplate)
		if err != nil {
			return options, fmt.Errorf("reading payload template %w", err)
		}

		value := string(b)
		options.PayloadTemplate = &value
	}

	if cmd.Flag("events").Changed {
		events, err := cmd.Flags().GetStringArray("events")
		if err != nil {
			return options, err
		}

		var eventTypes []testkube.EventType
		for _, event := range events {
			eventTypes = append(eventTypes, testkube.EventType(event))
		}

		options.Events = &eventTypes
	}

	if cmd.Flag("label").Changed {
		labels, err := cmd.Flags().GetStringToString("label")
		if err != nil {
			return options, err
		}

		options.Labels = &labels
	}

	if cmd.Flag("header").Changed {
		headers, err := cmd.Flags().GetStringToString("header")
		if err != nil {
			return options, err
		}

		options.Headers = &headers
	}

	return options, nil
}
