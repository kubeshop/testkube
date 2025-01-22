package webhooktemplates

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	webhooktemplatesmapper "github.com/kubeshop/testkube/pkg/mapper/webhooktemplates"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewCreateWebhookTemplateOptionsFromFlags creates create webhook template options from command flags
func NewCreateWebhookTemplateOptionsFromFlags(cmd *cobra.Command) (options apiv1.CreateWebhookTemplateOptions, err error) {
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
		config, err = common.GetWebhookConfig(configs)
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
		parameter, err = common.GetWebhookParameters(parameters)
		if err != nil {
			return options, err
		}
	}

	options = apiv1.CreateWebhookTemplateOptions{
		Name:                     name,
		Namespace:                namespace,
		Events:                   webhooktemplatesmapper.MapStringArrayToCRDEvents(events),
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
	}

	return options, nil
}

// NewUpdateWebhookTemplateOptionsFromFlags creates update webhook template options from command flags
func NewUpdateWebhookTemplateOptionsFromFlags(cmd *cobra.Command) (options apiv1.UpdateWebhookTemplateOptions, err error) {
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

		values, err := common.GetWebhookConfig(configs)
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

		values, err := common.GetWebhookParameters(parameters)
		if err != nil {
			return options, err
		}
		options.Parameters = &values
	}

	return options, nil
}
