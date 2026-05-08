package common

import (
	"encoding/csv"
	"errors"
	"strconv"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func GetWebhookConfig(configs map[string]string) (map[string]testkube.WebhookConfigValue, error) {
	config := make(map[string]testkube.WebhookConfigValue)
	for key, value := range configs {
		switch {
		case strings.HasPrefix(value, "value="):
			config[key] = testkube.WebhookConfigValue{
				Value: &testkube.BoxedString{Value: strings.TrimPrefix(value, "value=")},
			}
		case strings.HasPrefix(value, "secret="):
			data := strings.TrimPrefix(value, "secret=")
			r := csv.NewReader(strings.NewReader(data))
			r.Comma = ';'
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
				Secret: &testkube.SecretRef{
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

func GetWebhookParameters(parameters map[string]string) ([]testkube.WebhookParameterSchema, error) {
	parameter := make([]testkube.WebhookParameterSchema, 0)
	for key, value := range parameters {
		r := csv.NewReader(strings.NewReader(value))
		r.Comma = ';'
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

		parameter = append(parameter, testkube.WebhookParameterSchema{
			Name:        key,
			Description: records[0][0],
			Required:    required,
			Example:     records[0][2],
			Default_: &testkube.BoxedString{
				Value: records[0][3],
			},
			Pattern: records[0][4],
		})
	}

	return parameter, nil
}
