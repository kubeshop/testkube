package webhooks

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

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
		config, err = getWebhookConfig(configs)
		if err != nil {
			return options, err
		}
	}

	var parameter map[string]testkube.WebhookParameterSchema
	parameters, err := cmd.Flags().GetStringToString("parameter")
	if err != nil {
		return options, err
	}

	if len(parameters) != 0 {
		parameter, err = getWebhookParameters(parameters)
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

		values, err := getWebhookConfig(configs)
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

		values, err := getWebhookParameters(parameters)
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

	return options, nil
}

func getWebhookConfig(configs map[string]string) (map[string]testkube.WebhookConfigValue, error) {
	config := map[string]testkube.WebhookConfigValue{}
	for key, value := range configs {
		switch {
		case strings.HasPrefix(value, "public="):
			config[key] = testkube.WebhookConfigValue{
				Public: &testkube.BoxedString{Value: strings.TrimPrefix(value, "public=")},
			}
		case strings.HasPrefix(value, "private="):
			data := strings.TrimPrefix(value, "private=")
			r := csv.NewReader(strings.NewReader(data))
			r.Comma = ','
			r.LazyQuotes = true
			r.TrimLeadingSpace = true

			records, err := r.ReadAll()
			if err != nil {
				return nil, err
			}

			if len(records) != 1 {
				return nil, errors.New("single string expected")
			}

			if len(records[0]) != 3 {
				return nil, errors.New("3 fields expected")
			}

			config[key] = testkube.WebhookConfigValue{
				Private: &testkube.SecretRef{
					Namespace: records[0][0],
					Name:      records[0][1],
					Key:       records[0][2],
				},
			}
		default:
			continue
		}
	}

	return config, nil
}

func getWebhookParameters(parameters map[string]string) (map[string]testkube.WebhookParameterSchema, error) {
	parameter := map[string]testkube.WebhookParameterSchema{}
	for key, value := range parameters {
		r := csv.NewReader(strings.NewReader(value))
		r.Comma = ','
		r.LazyQuotes = true
		r.TrimLeadingSpace = true

		records, err := r.ReadAll()
		if err != nil {
			return nil, err
		}

		if len(records) != 1 {
			return nil, errors.New("single string expected")
		}

		if len(records[0]) != 5 {
			return nil, errors.New("5 fields expected")
		}

		var required bool
		required, err = strconv.ParseBool(records[0][1])
		if err != nil {
			return nil, err
		}

		parameter[key] = testkube.WebhookParameterSchema{
			Description: records[0][0],
			Required:    required,
			Example:     records[0][2],
			Default_: &testkube.BoxedString{
				Value: records[0][3],
			},
			Pattern: records[0][4],
		}
	}

	return parameter, nil
}
