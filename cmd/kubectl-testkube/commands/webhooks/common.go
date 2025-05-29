package webhooks

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	cmdcommon "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/internal/common"
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

	disabled, err := cmd.Flags().GetBool("disable")
	if err != nil {
		return options, err
	}

	payloadTemplateReference := cmd.Flag("payload-template-reference").Value.String()
	var config map[string]testkube.WebhookConfigValue
	configs, err := cmd.Flags().GetStringToString("config")
	if err != nil {
		return options, err
	}

	if len(configs) != 0 {
		config, err = cmdcommon.GetWebhookConfig(configs)
		if err != nil {
			return options, err
		}
	}

	var parameter []testkube.WebhookParameterSchema
	parameters, err := cmd.Flags().GetStringToString("parameter")
	if err != nil {
		return options, err
	}

	if len(parameters) != 0 {
		parameter, err = cmdcommon.GetWebhookParameters(parameters)
		if err != nil {
			return options, err
		}
	}

	var webhookTemplateReference *testkube.WebhookTemplateRef
	if cmd.Flag("webhook-template-reference").Changed {
		webhookTemplateReference = &testkube.WebhookTemplateRef{
			Name: cmd.Flag("webhook-template-reference").Value.String(),
		}
	}

	attachJunitSummary, err := cmd.Flags().GetBool("attach-junit-summary")
	if err != nil {
		return options, err
	}

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
		Disabled:                 disabled,
		Config:                   config,
		Parameters:               parameter,
		WebhookTemplateRef:       webhookTemplateReference,
		AttachJunitSummary:       attachJunitSummary,
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

	if cmd.Flag("disable").Changed {
		disabled, err := cmd.Flags().GetBool("disable")
		if err != nil {
			return options, err
		}
		options.Disabled = &disabled
	}

	if cmd.Flag("config").Changed {
		configs, err := cmd.Flags().GetStringToString("config")
		if err != nil {
			return options, err
		}

		values, err := cmdcommon.GetWebhookConfig(configs)
		if err != nil {
			return options, err
		}
		options.Config = &values
	}

	if cmd.Flag("parameter").Changed {
		parameters, err := cmd.Flags().GetStringToString("parameter")
		if err != nil {
			return options, err
		}

		values, err := cmdcommon.GetWebhookParameters(parameters)
		if err != nil {
			return options, err
		}
		options.Parameters = &values
	}

	if cmd.Flag("webhook-template-reference").Changed {
		options.WebhookTemplateRef = common.Ptr(&testkube.WebhookTemplateRef{
			Name: cmd.Flag("webhook-template-reference").Value.String(),
		})
	}

	if cmd.Flag("attach-jnit-summary").Changed {
		attachJunitSummary, err := cmd.Flags().GetBool("attach-jnit-summary")
		if err != nil {
			return options, err
		}
		options.AttachJunitSummary = &attachJunitSummary
	}

	return options, nil
}
